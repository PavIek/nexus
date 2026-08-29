package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func main() {
	conn, _ := net.Dial("tcp", "localhost:8080")
	defer conn.Close()

	msg := []byte("hello")
	binary.Write(conn, binary.BigEndian, uint32(len(msg)))
	conn.Write(msg)

	msg = []byte("world")
	binary.Write(conn, binary.BigEndian, uint32(len(msg)))
	conn.Write(msg)

	fmt.Println("request completed")
}
