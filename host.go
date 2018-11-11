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

	"github.com/kr/pty"
	"github.com/mitchellh/colorstring"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	ptmx   *os.File
	hostDc *webrtc.RTCDataChannel
)

type HostConfig struct {
	bashArgs []string
	tmux     bool
	oneWay   bool
}

func hostDataChannelOnOpen(dc *webrtc.RTCDataChannel, errChan chan error) func() {
	return func() {
		colorstring.Println("[bold]Terminal session started:")
		clearTerminal()

		cmd := exec.Command("bash", "-l")
		var err error
		ptmx, err = pty.Start(cmd)
		if err != nil {
			log.Println(err)
			errChan <- err
			return
		}

		if _, err = terminal.MakeRaw(int(os.Stdin.Fd())); err != nil {
			log.Println(err)
			errChan <- err
			return
		}
		go func() {
			if _, err = io.Copy(ptmx, os.Stdin); err != nil {
				log.Println(err)
			}
		}()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				log.Println("Sigint")
				errChan <- errors.New("sigint")
			}
		}()

		buf := make([]byte, 1024)
		for {
			nr, err := ptmx.Read(buf)
			if err != nil {
				if err == io.EOF {
					err = nil
				} else {
					log.Println(err)
				}
				errChan <- err
				return
			}
			if _, err = os.Stdout.Write(buf[0:nr]); err != nil {
				log.Println(err)
				errChan <- err
				return
			}
			if err = dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]}); err != nil {
				log.Println(err)
				errChan <- err
				return
			}
		}
	}
}

func hostDataChannelOnMessage(errChan chan error) func(payload datachannel.Payload) {
	return func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			data := string(p.Data)
			if data == "quit" {
				errChan <- nil
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
					_, err := ptmx.Write([]byte(msg[1]))
					if err != nil {
						log.Println(err)
						// errChan <- err
					}
					return
				}
				if msg[0] == "set_size" {
					var size []int
					_ = json.Unmarshal(p.Data, &size)
					ws, err := pty.GetsizeFull(ptmx)
					if err != nil {
						log.Println(err)
						errChan <- err
					}
					ws.Rows = uint16(size[1])
					ws.Cols = uint16(size[2])

					if len(size) >= 5 {
						ws.X = uint16(size[3])
						ws.Y = uint16(size[4])
					}

					if err = pty.Setsize(ptmx, ws); err != nil {
						log.Println(err)
						errChan <- err
					}
					return
				}
			}
			errChan <- errors.New(fmt.Sprintf(`Unmatched string message: "%s"`, string(p.Data)))
		case *datachannel.PayloadBinary:
			_, err := ptmx.Write(p.Data)
			if err != nil {
				log.Println(err)
				errChan <- err
			}
		default:
			log.Printf(
				"Message with type %s from DataChannel has no payload",
				p.PayloadType().String(),
			)
		}
	}
}

func hostOnDataChannel(errChan chan error) func(dc *webrtc.RTCDataChannel) {
	return func(dc *webrtc.RTCDataChannel) {
		hostDc = dc
		dc.Lock()
		defer dc.Unlock()
		dc.OnOpen = hostDataChannelOnOpen(dc, errChan)
		dc.Onmessage = hostDataChannelOnMessage(errChan)
	}
}

func mustReadStdin() (string, error) {
	var input string
	fmt.Scanln(&input)
	sd, err := decodeOffer(input)
	return sd.Sdp, err
}

func runHost(oneWay bool) (err error) {
	colorstring.Printf("[bold]Setting up a WebRTTY connection.\n\n")
	if oneWay {
		colorstring.Printf(
			"Warning: One-way connections rely on a third party to connect. " +
				"More info here: https://github.com/maxmcd/webrtty#one-way-connections\n\n")
	}
	pc, err := createPeerConnection()
	if err != nil {
		log.Println(err)
		return
	}
	errChan := make(chan error, 1)
	pc.OnDataChannel = hostOnDataChannel(errChan)

	// Create an offer to send to the browser
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Println(err)
		return
	}
	sessDesc := sessionDescription{
		Sdp: offer.Sdp,
	}
	if oneWay == true {
		sessDesc.TenKbSiteLoc = randSeq(100)
	}

	// Output the offer in base64 so we can paste it in browser
	colorstring.Printf("[bold]Connection ready. Here is your connection data:\n\n")
	fmt.Printf("%s\n\n", encodeOffer(sessDesc))
	colorstring.Printf(`[bold]Paste it in the terminal after the webrtty command` +
		"\n[bold]Or in a browser: [reset]https://maxmcd.github.io/webrtty/\n\n")

	var sd string
	if oneWay == false {
		colorstring.Println("[bold]When you have the answer, paste it below and hit enter:")
		// Wait for the answer to be pasted
		sd, err = mustReadStdin()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("Answer recieved, connecting...")
	} else {
		body, err := pollForResponse(sessDesc.TenKbSiteLoc)
		if err != nil {
			log.Println(err)
			return err
		}
		sessDesc, err := decodeOffer(body)
		if err != nil {
			log.Println(err)
			return err
		}
		sd = sessDesc.Sdp
	}

	// Set the remote SessionDescription
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  sd,
	}

	// Apply the answer as the remote description
	err = pc.SetRemoteDescription(answer)
	if err != nil {
		log.Println(err)
		return
	}

	oldTerminalState, err := terminal.GetState(int(os.Stdin.Fd()))
	if err != nil {
		log.Println(err)
		return
	}
	// Wait to quit
	err = <-errChan
	if hostDc != nil {
		// TODO: check dc state?
		hostDc.Send(datachannel.PayloadString{Data: []byte("quit")})
	}
	if err := terminal.Restore(int(os.Stdin.Fd()), oldTerminalState); err != nil {
		log.Println(err)
	}

	return err
}
