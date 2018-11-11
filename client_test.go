package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/kr/pty"
	"github.com/pions/webrtc/pkg/datachannel"
	"golang.org/x/crypto/ssh/terminal"
)

func TestClientDataChannelOnMessage(t *testing.T) {
	cc := clientConfig{}
	cc.errChan = make(chan error, 1)
	cc.oldTerminalState = &terminal.State{}
	onMessage := cc.dataChannelOnMessage()
	quitPayload := datachannel.PayloadString{Data: []byte("quit")}
	onMessage(&quitPayload)

	select {
	case err := <-cc.errChan:
		if err != nil {
			t.Error(err)
		}
	default:
		t.Error(errors.New("should return errChan"))
	}

	stdoutMock := tmpFile()
	stdout := os.Stdout
	os.Stdout = stdoutMock
	binaryPayload := datachannel.PayloadBinary{Data: []byte("s")}
	onMessage(&binaryPayload)
	os.Stdout = stdout
	stdoutMock.Seek(0, 0)
	msg, _ := ioutil.ReadAll(stdoutMock)
	if string(msg) != "s" {
		t.Error("bytes not written to stdout")
	}

}

func TestSendTermSize(t *testing.T) {
	hc := hostConfig{ptmxReady: true}
	c := exec.Command("sh")
	var err error
	hc.ptmx, err = pty.Start(c)
	if err != nil {
		t.Error(err)
	}
	if err := pty.Setsize(hc.ptmx, &pty.Winsize{
		Rows: 19,
		Cols: 29,
		X:    9,
		Y:    8,
	}); err != nil {
		t.Error(err)
	}

	dcSend := func(payload datachannel.Payload) error {
		switch p := payload.(type) {
		case datachannel.PayloadString:
			onMessage, hc := makeShPty(t)
			size, err := pty.GetsizeFull(hc.ptmx)
			if err != nil {
				t.Error(err)
			}
			if fmt.Sprintf("%v", size) != "&{0 0 0 0}" {
				t.Error("wrong size", size)
			}
			onMessage(&datachannel.PayloadString{Data: p.Data})
			select {
			case err := <-hc.errChan:
				if err != nil {
					t.Error(err)
				}
			default:

			}
			size, err = pty.GetsizeFull(hc.ptmx)
			if err != nil {
				t.Error(err)
			}
			if fmt.Sprintf("%v", size) != "&{19 29 9 8}" {
				t.Error("wrong size", size)
			}
		default:
			fmt.Println(p.PayloadType().String())
			t.Error("Should have matched")
		}
		return nil
	}
	if err := sendTermSize(hc.ptmx, dcSend); err != nil {
		t.Error(err)
	}

}
