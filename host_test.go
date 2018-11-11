package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/kr/pty"
	"github.com/pions/webrtc/pkg/datachannel"
)

func TestHosttDataChannelOnMessage(t *testing.T) {
	hc := hostConfig{ptmxReady: true}
	hc.errChan = make(chan error, 1)
	onMessage := hc.dataChannelOnMessage()
	quitPayload := datachannel.PayloadString{Data: []byte("quit")}
	onMessage(&quitPayload)

	select {
	case err := <-hc.errChan:
		if err != nil {
			t.Error(err)
		}
	default:
		t.Error(errors.New("should return errChan"))
	}

	stdoutMock := tmpFile()
	hc.ptmx = stdoutMock

	binaryPayload := datachannel.PayloadBinary{Data: []byte("s")}
	onMessage(&binaryPayload)
	stdoutMock.Seek(0, 0)
	msg, _ := ioutil.ReadAll(stdoutMock)
	if string(msg) != "s" {
		t.Error("bytes not written to stdout")
	}

}

func makeShPty(t *testing.T) (func(p datachannel.Payload), hostConfig) {
	hc := hostConfig{ptmxReady: true}
	hc.errChan = make(chan error, 1)
	onMessage := hc.dataChannelOnMessage()
	c := exec.Command("sh")
	var err error
	// redefine the global ptmx
	hc.ptmx, err = pty.Start(c)
	if err != nil {
		t.Error(err)
	}
	return onMessage, hc
}

func TestClientSetSizeOnMessage(t *testing.T) {
	onMessage, hc := makeShPty(t)

	sizeOnlyPayload := datachannel.PayloadString{Data: []byte(`["set_size", 20, 30]`)}
	onMessage(&sizeOnlyPayload)

	size, err := pty.GetsizeFull(hc.ptmx)
	if err != nil {
		t.Error(err)
	}
	if fmt.Sprintf("%v", size) != "&{20 30 0 0}" {
		t.Error("wrong size", size)
	}

	sizeAndCursorPayload := datachannel.PayloadString{Data: []byte(`["set_size", 20, 30, 10, 11]`)}
	onMessage(&sizeAndCursorPayload)

	size, err = pty.GetsizeFull(hc.ptmx)
	if err != nil {
		t.Error(err)
	}
	if fmt.Sprintf("%v", size) != "&{20 30 10 11}" {
		t.Error("wrong size", size)
	}
	select {
	case err := <-hc.errChan:
		if err != nil {
			t.Error(err)
		}
	default:

	}
}
