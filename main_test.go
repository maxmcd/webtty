package main

import (
	"log"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// func TestFull(t *testing.T) {
// 	hc := hostConfig{
// 		oneWay: false,
// 	}

// 	stdoutMock := tmpFile()
// 	// stdout := os.Stdout
// 	os.Stdout = stdoutMock

// 	go func() {
// 		if err := hc.run(); err != nil {
// 			t.Error(err)
// 		}
// 	}()

// 	for hc.offer.Sdp == "" {
// 		// wait for sdp to be set
// 		time.Sleep(1 * time.Millisecond)
// 	}

// 	cc := clientConfig{}
// 	go func() {
// 		if err := cc.runClient(encodeOffer(hc.offer)); err != nil {
// 			t.Error(err)
// 		}
// 	}()

// 	for cc.sessDesc.Sdp == "" {
// 		// wait for sdp to be set
// 		time.Sleep(1 * time.Millisecond)
// 	}
// 	stdoutMock.Write([]byte(encodeOffer))
// 	hc.answer = cc.sessDesc
// 	go hc.setHostRemoteDescriptionAndWait()

// }
