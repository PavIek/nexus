package main

import (
	"bytes"
	"fmt"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func handleRequestWithPool(payload []byte) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(payload)
	fmt.Println(buf.Len())
	bufPool.Put(buf)
}

func handleRequest(payload []byte) {
	buf := &bytes.Buffer{}
	buf.Write(payload)
	fmt.Println(buf.Len())
}
