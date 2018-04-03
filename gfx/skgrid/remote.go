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
	r := []byte{0}
	_, err = s.sock.Read(r)
	if err != nil {
		return err
	}
	switch r[0] {
	case 0x01:
		//log.Println("recevied remote ack")
	default:
		return fmt.Errorf("remote returned error code %2x", r[0])
	}
	return nil
}

func (s *Remote) Close() error {
	return s.sock.Close()
}
