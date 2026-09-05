package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"example.com/nexus/internal/hash"
	"golang.org/x/sys/unix"
)

const maxMessageSize = 4096

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, maxMessageSize)
	},
}

func main() {
	listenFD, err := listen(":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer unix.Close(listenFD)

	initWorkers()

	ep, err := NewEpoll()
	if err != nil {
		log.Fatal(err)
	}
	defer ep.Close()

	if err := ep.Add(listenFD); err != nil {
		log.Fatal(err)
	}

	conns := make(map[int]*Conn)
	states := make(map[int]*connState)

	for {
		events, err := ep.Wait()
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			log.Fatal(err)
		}

		for _, event := range events {
			fd := int(event.Fd)
			if fd == listenFD {
				acceptAll(ep, listenFD, conns, states)
				continue
			}

			conn := conns[fd]
			state := states[fd]
			if conn == nil || state == nil {
				continue
			}
			if err := readAvailable(conn, state); err != nil {
				closeConn(fd, conns, states)
			}
		}
	}
}

func listen(addr string) (int, error) {
	tcpAddr, err := tcpAddr(addr)
	if err != nil {
		return -1, err
	}

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return -1, err
	}
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if err := SetNonblock(fd); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if err := unix.Bind(fd, tcpAddr); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if err := unix.Listen(fd, unix.SOMAXCONN); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func tcpAddr(addr string) (*unix.SockaddrInet4, error) {
	switch addr {
	case ":8080":
		return &unix.SockaddrInet4{Port: 8080}, nil
	default:
		return nil, fmt.Errorf("unsupported listen address: %s", addr)
	}
}

func acceptAll(ep *Epoll, listenFD int, conns map[int]*Conn, states map[int]*connState) {
	for {
		fd, _, err := unix.Accept(listenFD)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				return
			}
			log.Print(err)
			continue
		}

		conn, err := NewConn(fd)
		if err != nil {
			unix.Close(fd)
			log.Print(err)
			continue
		}
		if err := ep.Add(conn.FD()); err != nil {
			conn.Close()
			log.Print(err)
			continue
		}

		conns[conn.FD()] = conn
		states[conn.FD()] = &connState{buf: bufferPool.Get().([]byte)}
	}
}

func closeConn(fd int, conns map[int]*Conn, states map[int]*connState) {
	if conn := conns[fd]; conn != nil {
		conn.Close()
		delete(conns, fd)
	}
	if state := states[fd]; state != nil {
		bufferPool.Put(state.buf)
		delete(states, fd)
	}
}

type connState struct {
	buf  []byte
	data []byte
}

func readAvailable(conn *Conn, state *connState) error {
	for {
		n, err := conn.Read(state.buf)
		if err != nil {
			if err == ErrWouldBlock {
				return nil
			}
			return err
		}

		state.data = append(state.data, state.buf[:n]...)
		if err := dispatchMessages(state); err != nil {
			return err
		}
	}
}

func dispatchMessages(state *connState) error {
	for {
		msg, ok, err := nextMessage(state)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		dispatch(msg)
	}
}

func nextMessage(state *connState) ([]byte, bool, error) {
	if len(state.data) < 4 {
		return nil, false, nil
	}

	length := binary.BigEndian.Uint32(state.data[:4])
	if length > maxMessageSize {
		return nil, false, fmt.Errorf("message too large: %d", length)
	}
	if uint32(len(state.data)-4) < length {
		return nil, false, nil
	}

	start := 4
	end := start + int(length)
	msg := append([]byte(nil), state.data[start:end]...)
	state.data = state.data[end:]
	return msg, true, nil
}

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
