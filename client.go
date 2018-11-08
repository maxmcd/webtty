package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/kr/pty"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

func sendTermSize(term *os.File, dcSend func(p datachannel.Payload) error) error {
	winSize, err := pty.GetsizeFull(term)
	if err != nil {
		glog.Fatal(err)
	}
	size := fmt.Sprintf(`["set_size",%d,%d,%d,%d]`,
		winSize.Rows, winSize.Cols, winSize.X, winSize.Y)

	return dcSend(&datachannel.PayloadString{Data: []byte(size)})
}

func clientDataChannelOnOpen(errChan chan error, dc *webrtc.RTCDataChannel) func() {
	return func() {
		fmt.Printf("Data channel '%s'-'%d'='%d' open.\n", dc.Label, dc.ID, *dc.MaxPacketLifeTime)
		oldTerminalState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			glog.Error(err)
			errChan <- err
		}
		defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldTerminalState) }() // Best effort.

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				sendTermSize(os.Stdin, dc.Send)
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.
		clearTerminal()
		buf := make([]byte, 1024)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				glog.Error(err)
				errChan <- err
			}
			err = dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
			if err != nil {
				glog.Error(err)
				errChan <- err
			}
		}
	}
}

func clientDataChannelOnMessage(errChan chan error, oldTerminalState *terminal.State) func(payload datachannel.Payload) {
	return func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			if string(p.Data) == "quit" {
				terminal.Restore(int(os.Stdin.Fd()), oldTerminalState)
				errChan <- nil
				return
			}
			errChan <- errors.New(fmt.Sprintf(`Unmatched string message: "%s"`, string(p.Data)))
		case *datachannel.PayloadBinary:
			f := bufio.NewWriter(os.Stdout)
			f.Write(p.Data)
			f.Flush()
			// fmt.Printf("Message '%s' from DataChannel '%s' payload '% 02x'\n", p.PayloadType().String(), dc.Label, p.Data)
		default:
			fmt.Printf("Message '%s' from DataChannel no payload \n", p.PayloadType().String())
		}
	}
}

func runClient(offerString string) (err error) {
	pc, err := createPeerConnection()
	if err != nil {
		glog.Error(err)
		return
	}
	// Set the remote SessionDescription
	maxPacketLifeTime := uint16(1000)
	var ordered bool = true
	dc, err := pc.CreateDataChannel("data", &webrtc.RTCDataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	})
	if err != nil {
		glog.Error(err)
		return
	}

	errChan := make(chan error, 1)
	oldTerminalState, err := terminal.GetState(int(os.Stdin.Fd()))
	if err != nil {
		glog.Error(err)
		return err
	}
	dc.Lock()
	dc.OnOpen = clientDataChannelOnOpen(errChan, dc)
	dc.Onmessage = clientDataChannelOnMessage(errChan, oldTerminalState)
	dc.Unlock()

	sdp, err := decodeOffer(offerString)
	if err != nil {
		glog.Error(err)
		return
	}
	offer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeOffer,
		Sdp:  sdp.Sdp,
	}

	if err = pc.SetRemoteDescription(offer); err != nil {
		glog.Error(err)
		return err
	}
	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		glog.Error(err)
		return
	}
	// Get the LocalDescription and take it to base64 so we can paste in browser

	fmt.Printf("Answer created. Send the following answer to the host:\n\n")
	fmt.Println(encodeOffer(answer.Sdp))
	return <-errChan
}
