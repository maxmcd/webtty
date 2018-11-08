package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreate10kbFile(t *testing.T) {
	path := randSeq(100)
	body := "secret"
	errorBody := randSeq(10*1000 + 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "/"+path != r.URL.String() {
			t.Errorf("wrong path: %s", r.URL)
		}
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}

		if len(bytes) > 10*1000 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("to long"))
			return
		}
		if string(bytes) != body {
			t.Error("body is wrong")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	tenKbUpUrl = ts.URL + "/"

	err := create10kbFile(path, body)
	if err != nil {
		t.Error(err)
	}

	err = create10kbFile(path, errorBody)
	if err == nil {
		t.Error("should have errored")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", http.StatusUnprocessableEntity)) {
		t.Error(err.Error())
	}
}

func TestRead10kbFile(t *testing.T) {
	path := randSeq(100)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "/not-found.txt" == r.URL.String() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if "/err" == r.URL.String() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if "/"+path != r.URL.String() {
			t.Errorf("wrong path: %s", r.URL)
		}
		w.Write([]byte("body"))
	}))
	tenKbUrl = ts.URL + "/"

	status, body, err := read10kbFile(path)
	if err != nil {
		t.Error(err)
	}
	if status != http.StatusOK {
		t.Error(status)
	}
	if body != "body" {
		t.Error(body)
	}

	status, body, err = read10kbFile("not-found.txt")
	if err != nil {
		t.Error(err)
	}
	if status != http.StatusNotFound {
		t.Error(status)
	}
	if body != "" {
		t.Error(body)
	}

	status, body, err = read10kbFile("err")
	if err == nil {
		t.Error("should have errored")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", http.StatusInternalServerError)) {
		t.Error(err.Error())
	}

}

func TestPollForResponse(t *testing.T) {
	var count int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count += 1
		if count < 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("body"))
	}))
	tenKbUrl = ts.URL + "/"

	body, err := pollForResponse("path")
	if body != "body" {
		t.Error(body)
	}
	if err != nil {
		t.Error(err)
	}

}
