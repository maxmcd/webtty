package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/kr/pty"
	"github.com/pion/webrtc/v2"
	"golang.org/x/crypto/ssh/terminal"
)

func TestClientDataChannelOnMessage(t *testing.T) {
	cs := clientSession{}
	cs.errChan = make(chan error, 1)
	cs.oldTerminalState = &terminal.State{}
	onMessage := cs.dataChannelOnMessage()
	quitPayload := webrtc.DataChannelMessage{IsString: true, Data: []byte("quit")}
	onMessage(quitPayload)

	select {
	case err := <-cs.errChan:
		if err != nil {
			t.Error(err)
		}
	default:
		t.Error(errors.New("should return errChan"))
	}

	stdoutMock := tmpFile()
	stdout := os.Stdout
	os.Stdout = stdoutMock
	binaryPayload := webrtc.DataChannelMessage{IsString: false, Data: []byte("s")}
	onMessage(binaryPayload)
	os.Stdout = stdout
	stdoutMock.Seek(0, 0)
	msg, _ := ioutil.ReadAll(stdoutMock)
	if string(msg) != "s" {
		t.Error("bytes not written to stdout")
	}

}

func TestSendTermSize(t *testing.T) {
	hs := hostSession{ptmxReady: true}
	c := exec.Command("sh")
	var err error
	hs.ptmx, err = pty.Start(c)
	if err != nil {
		t.Error(err)
	}
	if err := pty.Setsize(hs.ptmx, &pty.Winsize{
		Rows: 19,
		Cols: 29,
		X:    9,
		Y:    8,
	}); err != nil {
		t.Error(err)
	}

	dcSend := func(msg string) error {
		onMessage, hs := makeShPty(t)
		size, err := pty.GetsizeFull(hs.ptmx)
		if err != nil {
			t.Error(err)
		}
		if fmt.Sprintf("%v", size) != "&{0 0 0 0}" {
			t.Error("wrong size", size)
		}
		onMessage(webrtc.DataChannelMessage{IsString: true, Data: []byte(msg)})
		select {
		case err := <-hs.errChan:
			if err != nil {
				t.Error(err)
			}
		default:

		}
		size, err = pty.GetsizeFull(hs.ptmx)
		if err != nil {
			t.Error(err)
		}
		if fmt.Sprintf("%v", size) != "&{19 29 9 8}" {
			t.Error("wrong size", size)
		}
		return nil
	}
	if err := sendTermSize(hs.ptmx, dcSend); err != nil {
		t.Error(err)
	}

}
