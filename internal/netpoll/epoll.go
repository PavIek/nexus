package netpoll

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
