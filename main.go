package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
			glog.Fatal(err)
		}
	} else {
		err := runClient(offerString)
		if err != nil {
			glog.Fatal(err)
		}
	}

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
		// glog.Info("ICE Connection State has changed: %s\n", connectionState.String())
	}
	return
}

func runHost() (err error) {
	fmt.Println("Setting up WebRTTY connection.\n")

	pc, err := createPeerConnection()
	if err != nil {
		return
	}
	// Register data channel creation handling
	pc.OnDataChannel = func(d *webrtc.RTCDataChannel) {

		d.Lock()
		defer d.Unlock()
		cmd := exec.Command("bash")

		var ptmx *os.File
		// Register channel opening handling
		d.OnOpen = func() {
			// fmt.Printf("Data channel '%s'-'%d'-'%d' open. This is the host\n", d.Label, d.ID, d.MaxPacketLifeTime)
			fmt.Println("Data channel connected")
			var err error
			ptmx, err = pty.Start(cmd)

			if err != nil {
				glog.Fatal(err)
			}

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				for range c {
					d.Send(datachannel.PayloadString{Data: []byte("quit")})
					os.Exit(0)
				}
			}()

			buf := make([]byte, 1024)
			for {
				nr, err := ptmx.Read(buf)
				if err != nil {
					er := d.Send(datachannel.PayloadString{Data: []byte("quit")})
					glog.Error(er)
					glog.Fatal(err)
				}
				err = d.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
				if err != nil {
					glog.Fatal(err)
				}
			}

		}
		// stdinWriter := cmd.Stdin.(io.Writer)
		// Register message handling
		d.Onmessage = func(payload datachannel.Payload) {
			switch p := payload.(type) {
			case *datachannel.PayloadString:
				fmt.Printf("Message '%s' from DataChannel '%s' payload '%s'\n", p.PayloadType().String(), d.Label, string(p.Data))
				data := string(p.Data)
				if data[:4] == "size" {
					coords := strings.Split(data[5:], ",")
					out := [4]int{}
					fmt.Println(coords)
					for i, n := range coords {
						num, err := strconv.Atoi(n)
						check(err)
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
					}

				}
				// _, err := stdinWriter.Write(p.Data)
				// log.Println(err)
			case *datachannel.PayloadBinary:
				_, err := ptmx.Write(p.Data)
				if err != nil {
					log.Fatal(err)
				}
			default:
				fmt.Printf("Message '%s' from DataChannel '%s' no payload \n", p.PayloadType().String(), d.Label)
			}
		}
	}

	// Create an offer to send to the browser
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return
	}

	// Output the offer in base64 so we can paste it in browser
	fmt.Println("Connection ready. To connect to this session run:\n")
	fmt.Printf("webrtty %s\n\n", encodeOffer(offer.Sdp))
	fmt.Println("When you have the answer, paste it below and hit enter:")
	// Wait for the answer to be pasted
	sd := mustReadStdin()
	fmt.Println("Answer recieved, connecting...")

	// Set the remote SessionDescription
	answer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeAnswer,
		Sdp:  sd,
	}

	// Apply the answer as the remote description
	err = pc.SetRemoteDescription(answer)
	if err != nil {
		return
	}

	select {}
}

func runClient(offerString string) (err error) {
	pc, err := createPeerConnection()
	if err != nil {
		return
	}
	// Set the remote SessionDescription
	maxPacketLifeTime := uint16(1000)
	var ordered bool = true
	dataChannel, err := pc.CreateDataChannel("data", &webrtc.RTCDataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	})
	if err != nil {
		return
	}

	dataChannel.Lock()
	var oldState *terminal.State
	dataChannel.OnOpen = func() {
		fmt.Printf("Data channel '%s'-'%d'='%d' open.\n", dataChannel.Label, dataChannel.ID, *dataChannel.MaxPacketLifeTime)
		var err error
		oldState, err = terminal.MakeRaw(int(os.Stdin.Fd()))

		if err != nil {
			panic(err)
		}
		defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

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
				dataChannel.Send(datachannel.PayloadString{Data: []byte(size)})
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.

		buf := make([]byte, 1024)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				glog.Fatal(err)
			}
			err = dataChannel.Send(datachannel.PayloadBinary{Data: buf[0:nr]})
			if err != nil {
				glog.Fatal(err)
			}
		}
	}

	// Register the Onmessage to handle incoming messages
	dataChannel.Onmessage = func(payload datachannel.Payload) {
		switch p := payload.(type) {
		case *datachannel.PayloadString:
			fmt.Printf("Message '%s' from DataChannel '%s' payload '%s'\n", p.PayloadType().String(), dataChannel.Label, string(p.Data))
			if string(p.Data) == "quit" {
				terminal.Restore(int(os.Stdin.Fd()), oldState)
				os.Exit(0)
			}
		case *datachannel.PayloadBinary:
			f := bufio.NewWriter(os.Stdout)
			f.Write(p.Data)
			f.Flush()
			// fmt.Printf("Message '%s' from DataChannel '%s' payload '% 02x'\n", p.PayloadType().String(), dataChannel.Label, p.Data)
		default:
			fmt.Printf("Message '%s' from DataChannel '%s' no payload \n", p.PayloadType().String(), dataChannel.Label)
		}
	}

	dataChannel.Unlock()

	sdp, err := decodeOffer(offerString)
	if err != nil {
		return
	}
	offer := webrtc.RTCSessionDescription{
		Type: webrtc.RTCSdpTypeOffer,
		Sdp:  sdp,
	}

	if err = pc.SetRemoteDescription(offer); err != nil {
		return err
	}
	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return
	}
	// Get the LocalDescription and take it to base64 so we can paste in browser

	fmt.Println("Answer created. Send the following answer to the host:\n")
	fmt.Println(encodeOffer(answer.Sdp))
	select {}
}

// check is used to panic in an error occurs.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func encodeOffer(offer string) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(offer))
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func decodeOffer(offer string) (out string, err error) {
	sd, err := base64.StdEncoding.DecodeString(offer)
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

func mustReadStdin() string {
	var input string
	fmt.Scanln(&input)
	sd, err := decodeOffer(input)
	check(err)
	return sd
}
