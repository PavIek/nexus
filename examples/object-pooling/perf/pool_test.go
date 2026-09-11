package main

import (
	"bytes"
	"sync"
	"testing"
)

var requestPayload = bytes.Repeat([]byte("x"), 4096)

func BenchmarkWithoutPooling(b *testing.B) {
	for b.Loop() {
		buf := &bytes.Buffer{}
		buf.Write(requestPayload)
		_ = buf.Len()
	}
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func BenchmarkWithPooling(b *testing.B) {
	for b.Loop() {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Write(requestPayload)
		_ = buf.Bytes()
		bufPool.Put(buf)
	}
}
