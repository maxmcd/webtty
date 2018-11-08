package main

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	sdp := "something"
	offer := encodeOffer(sdp)
	sd, err := decodeOffer(offer)
	if err != nil {
		t.Error(err)
	}
	if sdp != sd.Sdp {
		t.Error("sdp doesn't match sdp")
	}

}
