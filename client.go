package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
	"github.com/mitchellh/colorstring"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

type clientConfig struct {
	dc               *webrtc.RTCDataChannel
	errChan          chan error
	isTerminal       bool
	oldTerminalState *terminal.State
	pc               *webrtc.RTCPeerConnection
	sessDesc         sessionDescription
}

func sendTermSize(term *os.File, dcSend func(p datachannel.Payload) error) error {
	winSize, err := pty.GetsizeFull(term)
	if err != nil {
		log.Fatal(err)
	}
	size := fmt.Sprintf(`["set_size",%d,%d,%d,%d]`,
		winSize.Rows, winSize.Cols, winSize.X, winSize.Y)

	return dcSend(datachannel.PayloadString{Data: []byte(size)})
}

func (cc *clientConfig) dataChannelOnOpen() func() {
	return func() {
		log.Printf("Data channel '%s'-'%d'='%d' open.\n", cc.dc.Label, cc.dc.ID, *cc.dc.MaxPacketLifeTime)
		colorstring.Println("[bold]Terminal session started:")
		oldTerminalState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			log.Println(err)
			cc.errChan <- err
		}
		defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldTerminalState) }() // Best effort.

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				err := sendTermSize(os.Stdin, cc.dc.Send)
				if err != nil {
					log.Println(err)
					cc.errChan <- err
				}
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.
		clearTerminal()
		buf := make([]byte, 1024)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				log.Println(err)
				cc.errChan <- err
			}
			err = cc.dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
			if err != nil {
				log.Println(err)
				cc.errChan <- err
			}
		}
	}
}

func (cc *clientConfig) dataChannelOnMessage() func(payload datachannel.Payload) {
	return func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			if string(p.Data) == "quit" {
				if cc.isTerminal {
					terminal.Restore(int(os.Stdin.Fd()), cc.oldTerminalState)
				}
				cc.errChan <- nil
				return
			}
			cc.errChan <- fmt.Errorf(`Unmatched string message: "%s"`, string(p.Data))
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

func (cc *clientConfig) runClient(offerString string) (err error) {
	cc.isTerminal = terminal.IsTerminal(int(os.Stdin.Fd()))
	if cc.pc, err = createPeerConnection(); err != nil {
		log.Println(err)
		return
	}
	// Set the remote SessionDescription
	maxPacketLifeTime := uint16(1000)
	ordered := true
	if cc.dc, err = cc.pc.CreateDataChannel("data", &webrtc.RTCDataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	}); err != nil {
		log.Println(err)
		return
	}

	cc.errChan = make(chan error, 1)

	if cc.isTerminal {
		if cc.oldTerminalState, err = terminal.GetState(int(os.Stdin.Fd())); err != nil {
			log.Println(err)
			return err
		}
	}

	cc.dc.Lock()
	cc.dc.OnOpen = cc.dataChannelOnOpen()
	cc.dc.Onmessage = cc.dataChannelOnMessage()
	cc.dc.Unlock()

	if cc.sessDesc, err = decodeOffer(offerString); err != nil {
		log.Println(err)
		return
	}
	offer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeOffer,
		Sdp:  cc.sessDesc.Sdp,
	}

	if err = cc.pc.SetRemoteDescription(offer); err != nil {
		log.Println(err)
		return err
	}
	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := cc.pc.CreateAnswer(nil)
	if err != nil {
		log.Println(err)
		return
	}
	encodedAnswer := encodeOffer(sessionDescription{Sdp: answer.Sdp})
	if cc.sessDesc.TenKbSiteLoc == "" {
		// Get the LocalDescription and take it to base64 so we can paste in browser
		fmt.Printf("Answer created. Send the following answer to the host:\n\n")
		fmt.Println(encodedAnswer)
	} else {
		if err := create10kbFile(cc.sessDesc.TenKbSiteLoc, encodedAnswer); err != nil {
			return err
		}
	}
	return <-cc.errChan
}
