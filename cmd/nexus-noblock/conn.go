package main

import (
	"errors"
	"io"

	"golang.org/x/sys/unix"
)

var ErrWouldBlock = errors.New("operation would block")

type Conn struct {
	fd int
}

func NewConn(fd int) (*Conn, error) {
	if err := SetNonblock(fd); err != nil {
		return nil, err
	}
	return &Conn{fd: fd}, nil
}

func (c *Conn) FD() int {
	return c.fd
}

func (c *Conn) Read(p []byte) (int, error) {
	n, err := unix.Read(c.fd, p)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			return 0, ErrWouldBlock
		}
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (c *Conn) Write(p []byte) (int, error) {
	n, err := unix.Write(c.fd, p)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			return 0, ErrWouldBlock
		}
		return 0, err
	}
	return n, nil
}

func (c *Conn) Close() error {
	err := unix.Close(c.fd)
	c.fd = -1
	return err
}

func SetNonblock(fd int) error {
	return unix.SetNonblock(fd, true)
}
