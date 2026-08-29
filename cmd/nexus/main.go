package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"example.com/nexus/internal/hash"
)

const maxMessageSize = 4096

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, maxMessageSize)
	},
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	initWorkers()

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

	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {

		msg, err := readMessageBuffer(conn, buf)
		if err != nil {
			return
		}

		dispatch(msg)
	}
}

func readMessageBuffer(conn net.Conn, buf []byte) ([]byte, error) {
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(buf[:4])
	if length > uint32(cap(buf)) {
		return nil, fmt.Errorf("message too large: %d", length)
	}

	data := buf[:length]
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

// func readMessage(conn net.Conn) ([]byte, error) {
// 	var length uint32
// 	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
// 		return nil, err
// 	}
// 	data := make([]byte, length)
// 	_, err := io.ReadFull(conn, data)
// 	return data, err
// }

func dispatch(msg []byte) {
	key := extractKey(msg)
	idx := hash.FNVNew32aHash(key) % len(workers)

	task := Task{
		Key:  key,
		Data: append([]byte(nil), msg...),
	}

	for !workers[idx].rb.Push(task) {
		fmt.Println("worker busy...")
		time.Sleep(10 * time.Second)
	}
}

func extractKey(msg []byte) string {
	return string(msg)
}
