package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var tenKbHost string = "https://up.10kb.site/"

func create10kbFile(path, body string) error {
	resp, err := http.Post(
		tenKbHost+path, "text/plain", bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf(
			"Resp %d 10kb.site error: %s", resp.StatusCode, string(body)))
	}
	return nil
}

func read10kbFile(path string) (int, string, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s", tenKbHost, path))
	if err != nil {
		return resp.StatusCode, "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusOK {
		return resp.StatusCode, "", errors.New(fmt.Sprintf(
			"Resp %d 10kb.site error: %s", resp.StatusCode, string(body)))
	}
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(body), nil
}

func pollForResponse(path string) (body string, err error) {
	var sc int
	for {
		sc, body, err = read10kbFile(path)
		if err != nil {
			return
		}
		if sc == http.StatusOK {
			return
		}
		if sc == http.StatusNotFound {
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func randSeq(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	// antiLig = []rune("acemnorsuvwxz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
