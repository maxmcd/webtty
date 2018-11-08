package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	fmt.Println(os.Args)
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
