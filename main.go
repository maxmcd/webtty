package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
)

func main() {
	oneWay := flag.Bool("o", false, "One-way connection with no response needed.")
	verbose := flag.Bool("v", false, "Verbose logging")
	nonInteractive := flag.Bool("non-interactive", false, "Set host to non-interactive")
	ni := flag.Bool("ni", false, "Set host to non-interactive")
	_ = flag.Bool("cmd", false, "The command to run. Default is \"bash -l\"\n"+
	    "Because this flag consumes the remainder of the command line,\n"+
	    "all other args (if present) must appear before this flag.\n"+
	    "eg: webrtty -o -v -ni -cmd docker run -it --rm alpine:latest sh")

	cmd := []string{"bash", "-l"}
	for i, arg := range os.Args {
		if arg == "-cmd" {
			cmd = os.Args[i+1:]
			os.Args = os.Args[:i]
		}
	}
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

	var err error
	if len(offerString) == 0 {
		hc := hostConfig{
			oneWay:         *oneWay,
			cmd:       cmd,
			nonInteractive: *nonInteractive || *ni,
		}
		err = hc.run()
	} else {
		cc := clientConfig{}
		err = cc.runClient(offerString)
	}
	if err != nil {
		fmt.Printf("Quitting with an unexpected error: \"%s\"\n", err)
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
