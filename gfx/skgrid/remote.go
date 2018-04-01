package skgrid

import (
	"fmt"
	"net"
)

type Remote struct {
	sock net.Conn
}

func NewRemote(addr string) (*Remote, error) {
	sock, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Remote{sock}, nil
}

func (s *Remote) Send(b []byte) error {
	n, err := s.sock.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("only wrote %d of %d bytes", n, len(b))
	}
	return nil
}

