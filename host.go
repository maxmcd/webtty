package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/kr/pty"
	"github.com/mitchellh/colorstring"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

type hostConfig struct {
	// mutex?
	bashArgs  []string
	tmux      bool
	oneWay    bool
	dc        *webrtc.RTCDataChannel
	pc        *webrtc.RTCPeerConnection
	sessDesc  sessionDescription
	errChan   chan error
	ptmx      *os.File
	ptmxReady bool
}

func (hc *hostConfig) dataChannelOnOpen() func() {
	return func() {
		colorstring.Println("[bold]Terminal session started:")
		clearTerminal()

		cmd := exec.Command("bash", "-l")
		var err error
		hc.ptmx, err = pty.Start(cmd)
		if err != nil {
			log.Println(err)
			hc.errChan <- err
			return
		}
		hc.ptmxReady = true

		if _, err = terminal.MakeRaw(int(os.Stdin.Fd())); err != nil {
			log.Println(err)
			hc.errChan <- err
			return
		}
		go func() {
			if _, err = io.Copy(hc.ptmx, os.Stdin); err != nil {
				log.Println(err)
			}
		}()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				log.Println("Sigint")
				hc.errChan <- errors.New("sigint")
			}
		}()

		buf := make([]byte, 1024)
		for {
			nr, err := hc.ptmx.Read(buf)
			if err != nil {
				if err == io.EOF {
					err = nil
				} else {
					log.Println(err)
				}
				hc.errChan <- err
				return
			}
			if _, err = os.Stdout.Write(buf[0:nr]); err != nil {
				log.Println(err)
				hc.errChan <- err
				return
			}
			if err = hc.dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]}); err != nil {
				log.Println(err)
				hc.errChan <- err
				return
			}
		}
	}
}

func (hc *hostConfig) dataChannelOnMessage() func(payload datachannel.Payload) {
	return func(payload datachannel.Payload) {

		for hc.ptmxReady != true {
			time.Sleep(1 * time.Millisecond)
		}

		switch p := payload.(type) {
		case *datachannel.PayloadString:
			data := string(p.Data)
			if data == "quit" {
				hc.errChan <- nil
				return
			}
			if len(data) > 2 && data[:2] == `["` {
				var msg []string
				_ = json.Unmarshal(p.Data, &msg)
				if msg[0] == "stdin" {
					toWrite := []byte(msg[1])
					if len(toWrite) == 0 {
						return
					}
					_, err := hc.ptmx.Write([]byte(msg[1]))
					if err != nil {
						log.Println(err)
						// hc.errChan <- err
					}
					return
				}
				if msg[0] == "set_size" {
					var size []int
					_ = json.Unmarshal(p.Data, &size)
					ws, err := pty.GetsizeFull(hc.ptmx)
					if err != nil {
						log.Println(err)
						hc.errChan <- err
						return
					}
					ws.Rows = uint16(size[1])
					ws.Cols = uint16(size[2])

					if len(size) >= 5 {
						ws.X = uint16(size[3])
						ws.Y = uint16(size[4])
					}

					if err := pty.Setsize(hc.ptmx, ws); err != nil {
						log.Println(err)
						hc.errChan <- err
					}
					return
				}
			}
			hc.errChan <- fmt.Errorf(`Unmatched string message: "%s"`, string(p.Data))
		case *datachannel.PayloadBinary:
			_, err := hc.ptmx.Write(p.Data)
			if err != nil {
				log.Println(err)
				hc.errChan <- err
			}
		default:
			log.Printf(
				"Message with type %s from DataChannel has no payload",
				p.PayloadType().String(),
			)
		}
	}
}

func (hc *hostConfig) onDataChannel() func(dc *webrtc.RTCDataChannel) {
	return func(dc *webrtc.RTCDataChannel) {
		hc.dc = dc
		dc.Lock()
		defer dc.Unlock()
		dc.OnOpen = hc.dataChannelOnOpen()
		dc.Onmessage = hc.dataChannelOnMessage()
	}
}

func mustReadStdin() (string, error) {
	var input string
	fmt.Scanln(&input)
	sd, err := decodeOffer(input)
	return sd.Sdp, err
}

func (hc *hostConfig) createSessDesc() (err error) {
	hc.pc, err = createPeerConnection()
	if err != nil {
		log.Println(err)
		return
	}
	hc.errChan = make(chan error, 1)
	hc.pc.OnDataChannel = hc.onDataChannel()

	// Create an offer to send to the browser
	offer, err := hc.pc.CreateOffer(nil)
	if err != nil {
		log.Println(err)
		return
	}
	hc.sessDesc = sessionDescription{
		Sdp: offer.Sdp,
	}
	if hc.oneWay == true {
		hc.sessDesc.TenKbSiteLoc = randSeq(100)
	}
	return
}

func (hc *hostConfig) run() (err error) {
	colorstring.Printf("[bold]Setting up a WebRTTY connection.\n\n")
	if hc.oneWay {
		colorstring.Printf(
			"Warning: One-way connections rely on a third party to connect. " +
				"More info here: https://github.com/maxmcd/webrtty#one-way-connections\n\n")
	}

	if err = hc.createSessDesc(); err != nil {
		return
	}

	// Output the offer in base64 so we can paste it in browser
	colorstring.Printf("[bold]Connection ready. Here is your connection data:\n\n")
	fmt.Printf("%s\n\n", encodeOffer(hc.sessDesc))
	colorstring.Printf(`[bold]Paste it in the terminal after the webrtty command` +
		"\n[bold]Or in a browser: [reset]https://maxmcd.github.io/webrtty/\n\n")

	if hc.oneWay == false {
		colorstring.Println("[bold]When you have the answer, paste it below and hit enter:")
		// Wait for the answer to be pasted
		hc.sessDesc.Sdp, err = mustReadStdin()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("Answer recieved, connecting...")
	} else {
		body, err := pollForResponse(hc.sessDesc.TenKbSiteLoc)
		if err != nil {
			log.Println(err)
			return err
		}
		hc.sessDesc, err = decodeOffer(body)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return hc.setHostRemoteDescriptionAndWait()
}

func (hc *hostConfig) setHostRemoteDescriptionAndWait() (err error) {
	// Set the remote SessionDescription
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  hc.sessDesc.Sdp,
	}

	// Apply the answer as the remote description
	if err = hc.pc.SetRemoteDescription(answer); err != nil {
		log.Println(err)
		return
	}

	oldTerminalState, err := terminal.GetState(int(os.Stdin.Fd()))
	if err != nil {
		log.Println(err)
		return
	}
	// Wait to quit
	err = <-hc.errChan
	if hc.dc != nil {
		// TODO: check dc state?
		hc.dc.Send(datachannel.PayloadString{Data: []byte("quit")})
	}
	if err := terminal.Restore(int(os.Stdin.Fd()), oldTerminalState); err != nil {
		log.Println(err)
	}
	return
}
