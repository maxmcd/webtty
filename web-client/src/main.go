package main

import (
	"syscall/js"

	"github.com/maxmcd/webrtty/pkg/sd"
)

func main() {
	c := make(chan struct{}, 0)
	registerCallbacks()
	<-c
}

var (
	key   string
	nonce string
)

func encode(i []js.Value) {
	encoded, err := func() (string, string) {
		answerSd := sd.SessionDescription{
			Sdp:   i[0].String(),
			Key:   key,
			Nonce: nonce,
		}
		// Encrypt with the shared keys from the offer
		if key != "" {
			if err := answerSd.Encrypt(); err != nil {
				return "", err.Error()
			}
		}

		// Don't upload the keys, the host has them
		answerSd.Key = ""
		answerSd.Nonce = ""
		return sd.Encode(answerSd), ""
	}()
	i[1].Invoke(encoded, err)
}

func decode(i []js.Value) {
	sdp, tkbsl, err := func() (string, string, string) {
		offer, err := sd.Decode(i[0].String())
		if err != nil {
			return "", "", err.Error()
		}
		if offer.Key != "" {
			key = offer.Key
			nonce = offer.Nonce
			if err := offer.Decrypt(); err != nil {
				return "", "", err.Error()
			}
		}
		return offer.Sdp, offer.TenKbSiteLoc, ""
	}()
	i[1].Invoke(sdp, tkbsl, err)
}

func registerCallbacks() {
	js.Global().Set("encode", js.NewCallback(encode))
	js.Global().Set("decode", js.NewCallback(decode))
}
