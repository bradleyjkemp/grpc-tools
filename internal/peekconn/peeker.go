package peekconn

import (
	"fmt"
	"net"
	"regexp"
)

type Peeker struct {
	net.Conn
	peeked []byte
}

type proxiedConnection interface {
	OriginalDestination() string
}

func (p *Peeker) OriginalDestination() string {
		switch underlying := p.Conn.(type) {
	case proxiedConnection:
		return underlying.OriginalDestination()
	default:
		return ""
	}
}

// Once called, the original connection *must not* be used
func (p *Peeker) PeekMatch(regexp *regexp.Regexp, len int) (bool, error) {
	if p.peeked != nil {
		return false, fmt.Errorf("have already peeked at this connection")
	}
	p.peeked = make([]byte, len)
	n, err := p.Conn.Read(p.peeked) // TODO: handle n < len(b)
	if err != nil {
		return false, err
	}
	if n != len {
		return false, fmt.Errorf("short conn reads aren't handled yet")
	}
	return regexp.Match(p.peeked), nil
}

func (p *Peeker) Read(b []byte) (int, error) {
	switch {
	case p.peeked != nil && len(p.peeked) <= len(b):
		copy(b, p.peeked)
		n := len(p.peeked)
		p.peeked = nil
		return n, nil

	case p.peeked != nil && len(p.peeked) > len(b):
		copy(b, p.peeked)
		p.peeked = p.peeked[len(b):]
		return len(b), nil

	default:
		return p.Conn.Read(b)
	}
}
