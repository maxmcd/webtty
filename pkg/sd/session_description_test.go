package sd

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	sdp := "something"
	offer := Encode(SessionDescription{Sdp: sdp})
	sd, err := Decode(offer)
	if err != nil {
		t.Error(err)
	}
	if sdp != sd.Sdp {
		t.Error("sdp doesn't match sdp")
	}
}

func TestEncryptAndDecrypt(t *testing.T) {
	sdp := "something"
	sd := SessionDescription{Sdp: sdp}
	sd.GenKeys()
	sd.Encrypt()
	if sd.Sdp == sdp {
		t.Error("should be encrypted")
	}
	offer := Encode(sd)
	sd, err := Decode(offer)
	if err != nil {
		t.Error(err)
	}
	err := sd.Decrypt()
	if err != nil {
		t.Error(err)
	}

	if sdp != sd.Sdp {
		t.Error("sdp doesn't match sdp")
	}

}
