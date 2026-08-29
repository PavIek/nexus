package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"example.com/nexus/internal/ringbuffer"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		msg, err := readMessage(conn)
		if err != nil {
			return
		}
		fmt.Println(string(msg))
	}
}

func readMessage(conn net.Conn) ([]byte, error) {
	var length uint32
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	data := make([]byte, length)
	_, err := io.ReadFull(conn, data)
	return data, err
}

func readMessageBuffer(conn net.Conn, buf []byte) ([]byte, error) {
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(buf[:4])
	if length > uint32(cap(buf)) {
		return nil, fmt.Errorf("message too large: %d", length)
	}

}

type Worker struct {
	rb *ringbuffer.RingBuffer[[]byte]
}

func (w *Worker) Run() {
	for {
		data, ok := w.rb.Pop()
		if !ok {
			continue
		}
		fmt.Println(data)
	}
}

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 4096)
	},
}
