package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io/ioutil"

	"github.com/btcsuite/btcutil/base58"
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
	return base58.Encode(b.Bytes())
}

type sessionDescription struct {
	Sdp          string
	TenKbSiteLoc string
}

func decodeOffer(offer string) (sd sessionDescription, err error) {
	sdCompressed := base58.Decode(offer)
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
