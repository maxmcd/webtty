package sd

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	"github.com/btcsuite/btcutil/base58"
)

func Encode(offer SessionDescription) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(offer.Sdp))
	w.Close()
	offer.Sdp = base58.Encode(b.Bytes())
	offerBytes, _ := json.Marshal(offer)
	return base58.Encode(offerBytes)
}

func Decode(offer string) (sd SessionDescription, err error) {
	decodeBytes := base58.Decode(offer)
	if err = json.Unmarshal(decodeBytes, &sd); err != nil {
		return
	}
	var b bytes.Buffer
	b.Write(base58.Decode(sd.Sdp))
	r, err := zlib.NewReader(&b)
	if err != nil {
		return
	}
	deflateBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	sd.Sdp = string(deflateBytes)
	return
}

type SessionDescription struct {
	Sdp          string
	TenKbSiteLoc string
	Key          string
	Nonce        string
}

func (sd *SessionDescription) GenKeys() (err error) {
	key := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, key); err != nil {
		return
	}

	sd.Key = hex.EncodeToString(key)

	nonce := make([]byte, 12)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}
	sd.Nonce = hex.EncodeToString(nonce)
	return
}

func (sd *SessionDescription) Encrypt() (err error) {

	nonce, err := hex.DecodeString(sd.Nonce)
	if err != nil {
		return
	}

	key, err := hex.DecodeString(sd.Key)
	if err != nil {
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	sd.Sdp = hex.EncodeToString(aesgcm.Seal(nil, nonce, []byte(sd.Sdp), nil))
	return
}

func (sd *SessionDescription) Decrypt() (err error) {
	key, err := hex.DecodeString(sd.Key)
	if err != nil {
		log.Println(err)
		return
	}
	ciphertext, err := hex.DecodeString(sd.Sdp)
	if err != nil {
		log.Println(err)
		return
	}
	nonce, err := hex.DecodeString(sd.Nonce)
	if err != nil {
		log.Println(err)
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err)
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Println(err)
		return
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sd.Sdp = string(plaintext)
	return
}
