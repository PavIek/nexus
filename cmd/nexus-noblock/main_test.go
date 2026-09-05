package main

import (
	"encoding/binary"
	"testing"
)

func TestNextMessageReadsMultipleFrames(t *testing.T) {
	state := &connState{}
	state.data = appendFrame(state.data, "hello")
	state.data = appendFrame(state.data, "world")

	msg, ok, err := nextMessage(state)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || string(msg) != "hello" {
		t.Fatalf("first message = %q, ok=%v", msg, ok)
	}

	msg, ok, err = nextMessage(state)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || string(msg) != "world" {
		t.Fatalf("second message = %q, ok=%v", msg, ok)
	}

	_, ok, err = nextMessage(state)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected complete message")
	}
}

func TestNextMessageWaitsForPartialFrame(t *testing.T) {
	state := &connState{}
	state.data = appendFrame(state.data, "hello")[:6]

	_, ok, err := nextMessage(state)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected complete message")
	}
}

func appendFrame(dst []byte, msg string) []byte {
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(msg)))
	dst = append(dst, header[:]...)
	return append(dst, msg...)
}
