package main

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

type config struct {
	oldTerminalState *terminal.State
}

func (hc *hostConfig) restoreTerminalState() error {
	if hc.oldTerminalState != nil {
		return terminal.Restore(int(os.Stdin.Fd()), hc.oldTerminalState)
	}
	return nil
}

func (hc *hostConfig) makeRawTerminal() error {
	var err error
	hc.oldTerminalState, err = terminal.MakeRaw(int(os.Stdin.Fd()))
	return err
}
