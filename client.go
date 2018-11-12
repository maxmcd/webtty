package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
	"github.com/maxmcd/webrtty/pkg/sd"
	"github.com/mitchellh/colorstring"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

type clientSession struct {
	session
	dc          *webrtc.RTCDataChannel
	offerString string
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

func (cs *clientSession) dataChannelOnOpen() func() {
	return func() {
		log.Printf("Data channel '%s'-'%d'='%d' open.\n", cs.dc.Label, cs.dc.ID, *cs.dc.MaxPacketLifeTime)
		colorstring.Println("[bold]Terminal session started:")

		if err := cs.makeRawTerminal(); err != nil {
			log.Println(err)
			cs.errChan <- err
		}

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				err := sendTermSize(os.Stdin, cs.dc.Send)
				if err != nil {
					log.Println(err)
					cs.errChan <- err
				}
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.
		buf := make([]byte, 1024)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				log.Println(err)
				cs.errChan <- err
			}
			err = cs.dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
			if err != nil {
				log.Println(err)
				cs.errChan <- err
			}
		}
	}
}

func (cs *clientSession) dataChannelOnMessage() func(payload datachannel.Payload) {
	return func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			if string(p.Data) == "quit" {
				if cs.isTerminal {
					terminal.Restore(int(os.Stdin.Fd()), cs.oldTerminalState)
				}
				cs.errChan <- nil
				return
			}
			cs.errChan <- fmt.Errorf(`Unmatched string message: "%s"`, string(p.Data))
		case *datachannel.PayloadBinary:
			f := bufio.NewWriter(os.Stdout)
			f.Write(p.Data)
			f.Flush()
		default:
			cs.errChan <- fmt.Errorf(
				"Message with type %s from DataChannel has no payload",
				p.PayloadType().String())
		}
	}
}

func (cs *clientSession) run() (err error) {
	if err = cs.init(); err != nil {
		return
	}

	maxPacketLifeTime := uint16(1000) // Arbitrary
	ordered := true
	if cs.dc, err = cs.pc.CreateDataChannel("data", &webrtc.RTCDataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	}); err != nil {
		log.Println(err)
		return
	}

	cs.dc.Lock()
	cs.dc.OnOpen = cs.dataChannelOnOpen()
	cs.dc.Onmessage = cs.dataChannelOnMessage()
	cs.dc.Unlock()

	if cs.offer, err = sd.Decode(cs.offerString); err != nil {
		log.Println(err)
		return
	}
	if cs.offer.Key != "" {
		if err = cs.offer.Decrypt(); err != nil {
			log.Println(err)
			return
		}
	}
	offer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeOffer,
		Sdp:  cs.offer.Sdp,
	}

	if err = cs.pc.SetRemoteDescription(offer); err != nil {
		log.Println(err)
		return err
	}
	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := cs.pc.CreateAnswer(nil)
	if err != nil {
		log.Println(err)
		return
	}
	answerSd := sd.SessionDescription{
		Sdp:   answer.Sdp,
		Key:   cs.offer.Key,
		Nonce: cs.offer.Nonce,
	}
	if cs.offer.Key != "" {
		// Encrypt with the shared keys from the offer
		_ = answerSd.Encrypt()

		// Don't upload the keys, the host has them
		answerSd.Key = ""
		answerSd.Nonce = ""
	}

	encodedAnswer := sd.Encode(answerSd)
	if cs.offer.TenKbSiteLoc == "" {
		// Get the LocalDescription and take it to base64 so we can paste in browser
		fmt.Printf("Answer created. Send the following answer to the host:\n\n")
		fmt.Println(encodedAnswer)
	} else {
		if err := create10kbFile(cs.offer.TenKbSiteLoc, encodedAnswer); err != nil {
			return err
		}
	}
	err = <-cs.errChan
	cs.cleanup()
	return err
}
