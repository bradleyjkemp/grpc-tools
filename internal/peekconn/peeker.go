package peekconn

import (
	"fmt"
	"net"
	"regexp"
	"sync"
)

type tcpLike interface {
	CloseRead() error
	CloseWrite() error
}

type peeker struct {
	net.Conn
	wg     sync.WaitGroup // allows for supporting CloseRead and CloseWrite if the underlying conn doesn't
	peeked []byte
}

func New(underlying net.Conn) *peeker {
	p := &peeker{
		Conn: underlying,
		wg:   sync.WaitGroup{},
	}
	p.wg.Add(2)
	return p
}

func (p *peeker) CloseRead() error {
	switch underlying := p.Conn.(type) {
	case tcpLike:
		return underlying.CloseRead()
	default:
		p.wg.Done()
		return nil
	}
}

func (p *peeker) CloseWrite() error {
	switch underlying := p.Conn.(type) {
	case tcpLike:
		return underlying.CloseWrite()
	default:
		p.wg.Done()
		p.wg.Wait() // arbitrarily chosen that CloseWrite closes the underlying and CloseRead is a noop
		return p.Conn.Close()
	}
}

type proxiedConnection interface {
	OriginalDestination() string
}

func (p *peeker) OriginalDestination() string {
	switch underlying := p.Conn.(type) {
	case proxiedConnection:
		return underlying.OriginalDestination()
	default:
		return ""
	}
}

// Once called, the original connection *must not* be used
func (p *peeker) PeekMatch(regexp *regexp.Regexp, len int) (bool, error) {
	if p.peeked != nil {
		return false, fmt.Errorf("have already peeked at this connection")
	}
	p.peeked = make([]byte, len)
	n, err := p.Conn.Read(p.peeked) // TODO: handle n < len(b)
	if err != nil {
		return false, err
	}
	if n != len {
		return false, fmt.Errorf("short conn reads aren't handled yet, wanted %d bytes, got %d", len, n)
	}
	return regexp.Match(p.peeked), nil
}

func (p *peeker) Read(b []byte) (int, error) {
	switch {
	case p.peeked != nil && len(p.peeked) <= len(b):
		// reading more than the peeked buffer
		copy(b, p.peeked)
		n, err := p.Conn.Read(b[len(p.peeked):])
		n += len(p.peeked)
		p.peeked = nil
		return n, err

	case p.peeked != nil && len(p.peeked) > len(b):
		// reading some of the peeked buffer
		copy(b, p.peeked)
		p.peeked = p.peeked[len(b):]
		return len(b), nil

	default:
		// buffer already read completely
		return p.Conn.Read(b)
	}
}
