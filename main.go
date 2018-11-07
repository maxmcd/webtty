package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/kr/pty"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"github.com/pions/webrtc/pkg/ice"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	var offerString string
	if len(os.Args) > 1 {
		offerString = os.Args[1]
	}

	if len(offerString) == 0 {
		err := runHost()
		if err != nil {
			glog.Error(err)
		}
	} else {
		err := runClient(offerString)
		if err != nil {
			glog.Error(err)
		}
	}
	resumeTerminal()
}

func createPeerConnection() (pc *webrtc.RTCPeerConnection, err error) {
	config := webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	pc, err = webrtc.New(config)
	if err != nil {
		return
	}
	pc.OnICEConnectionStateChange = func(connectionState ice.ConnectionState) {
		glog.Infof("ICE Connection State has changed: %s\n", connectionState.String())
	}
	return
}

var ptmx *os.File

func hostDataChannelOnOpen(dc *webrtc.RTCDataChannel, errChan chan error) func() {
	return func() {

		// find the control sequence to resume terminal state from earlier
		clearTerminal()

		cmd := exec.Command("bash")
		var err error
		ptmx, err = pty.Start(cmd)

		if err != nil {
			glog.Error(err)
			errChan <- err
			return
		}

		_, err = terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			glog.Error(err)
			errChan <- err
			return
		}
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				dc.Send(datachannel.PayloadString{Data: []byte("quit")})
				glog.Error("Sigint")
				errChan <- nil
			}
		}()

		buf := make([]byte, 1024)
		for {
			nr, err := ptmx.Read(buf)
			if err != nil {
				// TODO: check for EOF
				glog.Error(err)
				errChan <- err
				return
			}
			os.Stdout.Write(buf[0:nr])
			err = dc.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
			if err != nil {
				glog.Error(err)
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
			if len(data) > 4 && data[:4] == "size" {
				coords := strings.Split(data[5:], ",")
				out := [4]int{}
				for i, n := range coords {
					num, err := strconv.Atoi(n)
					if err != nil {
						glog.Error(err)
						errChan <- err
					}
					out[i] = num
				}
				time.Sleep(time.Millisecond * 100)
				err := pty.Setsize(ptmx, &pty.Winsize{
					Rows: uint16(out[0]),
					Cols: uint16(out[1]),
					X:    uint16(out[2]),
					Y:    uint16(out[3]),
				})
				if err != nil {
					glog.Error(err)
					// errChan <- err
				}
			}
			if len(data) > 2 && data[:2] == `["` {
				var msg []string
				_ = json.Unmarshal(p.Data, &msg)
				if msg[0] == "stdin" {
					_, err := ptmx.Write([]byte(msg[1]))
					if err != nil {
						glog.Error(err)
						errChan <- err
					}
				}
				if msg[0] == "set_size" {
					// TODO: error checking, everywhere
					var size []int
					_ = json.Unmarshal(p.Data, &size)
					ws, err := pty.GetsizeFull(ptmx)
					if err != nil {
						glog.Error(err)
						errChan <- err
					}
					ws.Rows = uint16(size[1])
					ws.Cols = uint16(size[2])
					err = pty.Setsize(ptmx, ws)
					if err != nil {
						glog.Error(err)
						errChan <- err
					}
				}
			}

		case *datachannel.PayloadBinary:
			_, err := ptmx.Write(p.Data)
			if err != nil {
				glog.Error(err)
				errChan <- err
			}
		default:
			fmt.Printf("Message '%s' from DataChannel '%s' no payload \n", p.PayloadType().String())
		}
	}
}

func hostOnDataChannel(errChan chan error) func(dc *webrtc.RTCDataChannel) {
	return func(dc *webrtc.RTCDataChannel) {
		hostDc = dc
		fmt.Println("GOT DATA CHANNEL")
		dc.Lock()
		defer dc.Unlock()
		dc.OnOpen = hostDataChannelOnOpen(dc, errChan)
		dc.Onmessage = hostDataChannelOnMessage(errChan)
	}
}

var hostDc *webrtc.RTCDataChannel

func runHost() (err error) {
	fmt.Println("Setting up WebRTTY connection.\n")

	pc, err := createPeerConnection()
	if err != nil {
		glog.Error(err)
		return
	}
	errChan := make(chan error, 1)
	pc.OnDataChannel = hostOnDataChannel(errChan)

	// Create an offer to send to the browser
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		glog.Error(err)
		return
	}

	// Output the offer in base64 so we can paste it in browser
	fmt.Println("Connection ready. To connect to this session run:\n")
	fmt.Printf("webrtty %s\n\n", encodeOffer(offer.Sdp))
	fmt.Println("When you have the answer, paste it below and hit enter:")
	// Wait for the answer to be pasted
	sd, err := mustReadStdin()
	if err != nil {
		glog.Error(err)
		return
	}
	fmt.Println("Answer recieved, connecting...")

	// Set the remote SessionDescription
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  sd,
	}

	// Apply the answer as the remote description
	err = pc.SetRemoteDescription(answer)
	if err != nil {
		glog.Error(err)
		return
	}

	oldTerminalState, err := terminal.GetState(int(os.Stdin.Fd()))
	if err != nil {
		glog.Error(err)
		return
	}
	// Wait to quit
	err = <-errChan
	if hostDc != nil {
		// TODO: check dc state?
		hostDc.Send(datachannel.PayloadString{Data: []byte("quit")})
	}
	if err := terminal.Restore(int(os.Stdin.Fd()), oldTerminalState); err != nil {
		glog.Error(err)
	}

	return err
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
				winSize, err := pty.GetsizeFull(os.Stdin)
				if err != nil {
					glog.Fatal(err)
				}
				size := fmt.Sprintf("size,%d,%d,%d,%d",
					winSize.Rows, winSize.Cols, winSize.X, winSize.Y)
				dc.Send(datachannel.PayloadString{Data: []byte(size)})
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
			fmt.Printf("Message '%s' from DataChannel payload '%s'\n", p.PayloadType().String(), string(p.Data))
			if string(p.Data) == "quit" {
				terminal.Restore(int(os.Stdin.Fd()), oldTerminalState)
				errChan <- nil
			}
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
		Sdp:  sdp,
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

	fmt.Println("Answer created. Send the following answer to the host:\n")
	fmt.Println(encodeOffer(answer.Sdp))
	return <-errChan
}

// check is used to panic in an error occurs.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func encodeOffer(offer string) string {
	fmt.Println(offer)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(offer))
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func decodeOffer(offer string) (out string, err error) {
	var sd []byte
	for i := 0; i < 2; i++ {
		sd, err = base64.StdEncoding.DecodeString(offer)
		if err != nil {
			// copy and paste is hard
			offer += "="
		}
	}
	if err != nil {
		return
	}
	var b bytes.Buffer
	b.Write(sd)
	r, err := zlib.NewReader(&b)
	if err != nil {
		return
	}
	deflateBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	out = string(deflateBytes)
	return
}

func mustReadStdin() (string, error) {
	var input string
	fmt.Scanln(&input)
	return decodeOffer(input)
}

func clearTerminal() {
	// http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x361.html
	// fmt.Print("\033[s")  // save cursor state
	fmt.Print("\033[2J") // clear terminal
	fmt.Print("\033[H")  // move cursor to 0,0
}

func resumeTerminal() {
	// http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x361.html
	// fmt.Print("\033[u")
}
