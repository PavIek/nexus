package main

import "golang.org/x/sys/unix"

type Epoll struct {
	fd int
}

func NewEpoll() (*Epoll, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Epoll{fd: fd}, nil
}

func (e *Epoll) Add(fd int) error {
	return unix.EpollCtl(e.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLET,
		Fd:     int32(fd),
	})
}

func (e *Epoll) Wait() ([]unix.EpollEvent, error) {
	events := make([]unix.EpollEvent, 128)
	n, err := unix.EpollWait(e.fd, events, -1)
	if err != nil {
		return nil, err
	}
	return events[:n], nil
}

func (e *Epoll) Close() error {
	return unix.Close(e.fd)
}
