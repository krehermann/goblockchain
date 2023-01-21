
import (
	"fmt"
	"net"

	"go.uber.org/zap"
)

type TcpTransport struct {
	addr   net.Addr
	logger *zap.Logger

	listener net.Listener
	net.Conn
	connections map[net.Addr]net.Conn
	errCh       chan error
}

type TcpTransportOpt func(t *TcpTransport) *TcpTransport

func TcpLogger(l *zap.Logger) TcpTransportOpt {
	return func(t *TcpTransport) *TcpTransport {
		t.logger = l
		return t
	}
}

func NewTcpTransport(addr net.Addr, opts ...TcpTransportOpt) (*TcpTransport, error) {
	t := &TcpTransport{
		addr:        addr,
		logger:      zap.L(),
		connections: make(map[net.Addr]net.Conn),
		errCh:       make(chan error),
	}

	for _, opt := range opts {
		t = opt(t)
	}
	t.logger = t.logger.Named(fmt.Sprintf("tcp-%s", addr.String()))

	l, err := net.Listen(t.addr.Network(), t.addr.String())
	if err != nil {
		return nil, err
	}
	t.listener = l
	go t.listen()
	return t, nil
}

func (t *TcpTransport) listen() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.errCh <- err
			continue
		}
		id, err := addrOf(conn)
		if err != nil {
			t.errCh <- err
			continue
		}
		t.connections[id] = conn
	}
}

func addrOf(c net.Conn) (net.Addr, error) {
	if c.LocalAddr() != nil {
		return c.LocalAddr(), nil
	}
	if c.RemoteAddr() != nil {
		return c.RemoteAddr(), nil
	}
	return nil, fmt.Errorf("connection %+v doesn't have a known address", c)
}
