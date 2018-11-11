package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
)

func main() {
	oneWay := flag.Bool("o", false, "One-way connection with no response needed.")
	verbose := flag.Bool("v", false, "Verbose logging")
	flag.Parse()
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	args := flag.Args()
	var offerString string
	if len(args) > 0 {
		offerString = args[len(args)-1]
	}

	if len(offerString) == 0 {
		err := runHost(*oneWay)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		err := runClient(offerString)
		if err != nil {
			fmt.Println(err)
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
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	}
	return
}

func clearTerminal() {
	// http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x361.html
	// fmt.Print("\033[s")  // save cursor state
	// fmt.Print("\033[2J") // clear terminal
	// fmt.Print("\033[H")  // move cursor to 0,0
}

func resumeTerminal() {
	// http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x361.html
	// fmt.Print("\033[u")
}
