package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
)

func encodeOffer(offer sessionDescription) string {
	bytes, _ := json.Marshal(offer)
	return encodeBytes(bytes)
}

func encodeBytes(offer []byte) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(offer)
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

type sessionDescription struct {
	Sdp          string
	TenKbSiteLoc string
}

func decodeOffer(offer string) (sd sessionDescription, err error) {
	var sdCompressed []byte
	for i := 0; i < 2; i++ {
		sdCompressed, err = base64.StdEncoding.DecodeString(offer)
		if err != nil {
			// copy and paste is hard
			offer += "="
		}
		// TODO: if we're going to do this retry thing, at least
		// check if the returning binary is text
	}
	if err != nil {
		return
	}
	var b bytes.Buffer
	b.Write(sdCompressed)
	r, err := zlib.NewReader(&b)
	if err != nil {
		return
	}
	deflateBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	if err = json.Unmarshal(deflateBytes, &sd); err != nil {
		return
	}

	return
}
