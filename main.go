package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	stunServer := flag.String("s", "stun:stun.l.google.com:19302", "The stun server to use")

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
		hc := hostSession{
			oneWay:         *oneWay,
			cmd:            cmd,
			nonInteractive: *nonInteractive || *ni,
		}
		hc.stunServers = []string{*stunServer}
		err = hc.run()
	} else {
		cc := clientSession{
			offerString: offerString,
		}
		cc.stunServers = []string{*stunServer}
		err = cc.run()
	}
	if err != nil {
		fmt.Printf("Quitting with an unexpected error: \"%s\"\n", err)
	}
}
