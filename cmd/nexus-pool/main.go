package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"example.com/nexus/internal/hash"
	"example.com/nexus/internal/shardmap"
)

const maxMessageSize = 4096

var (
	bufferPool = sync.Pool{
		New: func() any {
			return make([]byte, maxMessageSize)
		},
	}
	store = shardmap.New(64)
	pool  *Pool
)

func main() {
	pool = NewPool(runtime.NumCPU())
	defer pool.Close()

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

	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		msg, err := readMessageBuffer(conn, buf)
		if err != nil {
			return
		}

		task := Task{
			Key:  extractKey(msg),
			Data: append([]byte(nil), msg...),
		}
		for !pool.Submit(task) {
			fmt.Println("pool busy...")
			time.Sleep(10 * time.Millisecond)
		}
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

func processTask(task Task) {
	fmt.Println(fmt.Sprintf("process task: [%+v]", task), string(task.Data))
	store.Set(task.Key, int64(len(task.Data)))
}

func workerIndex(key string, numWorkers int) int {
	return int(hash.FNVNew32aHash(key) % numWorkers)
}

func extractKey(msg []byte) string {
	return string(msg)
}
