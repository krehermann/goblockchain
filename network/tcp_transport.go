package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
)

type TcpTransport struct {
	addr   net.Addr
	logger *zap.Logger

	listener net.Listener
	net.Conn

	lock        sync.RWMutex
	connections map[net.Addr]net.Conn

	recv chan RPC
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
		lock:        sync.RWMutex{},
		connections: make(map[net.Addr]net.Conn),
		recv:        make(chan RPC, 1024),
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

func (t *TcpTransport) Recv() <-chan RPC {
	return t.recv
}

func (t *TcpTransport) Get(addr net.Addr) (Transport, bool) {
	return nil, false
}
func (t *TcpTransport) Connect(o Transport) error {
	// has to be a tcp partner
	tcpPartner := o.(*TcpTransport)

	conn, err := net.Dial(o.Addr().Network(), o.Addr().String())
	if err != nil {
		return err
	}

	t.lock.Lock()
	_, exists := t.connections[tcpPartner.Addr()]
	if !exists {
		t.connections[tcpPartner.Addr()] = conn
	}
	t.lock.Unlock()

	return nil
}

func (t *TcpTransport) Broadcast(p Payload) error {
	panic("wtf")
}

func (t *TcpTransport) Send(to net.Addr, payload Payload) error {
	t.logger.Debug("send message",
		zap.String("from", string(t.addr.String())),
		zap.String("to", string(to.String())),
		zap.Int("msg length", len(payload.data)),
	)

	t.lock.RLock()
	peer, exists := t.connections[to]
	t.lock.RUnlock()

	if !exists {
		return fmt.Errorf("%s unknown peer %s", t.addr, to)
	}

	// need loop?
	n, err := peer.Write(payload.data)
	if err != nil {
		return err
	}
	if n != len(payload.data) {
		return fmt.Errorf("corrupt write len %d != %d", n, len(payload.data))
	}

	return nil

}

func (t *TcpTransport) read(c net.Conn) {
	for {

		var buf bytes.Buffer
		_, err := io.Copy(&buf, c)
		if err != nil {
			t.logger.Error("reading from conn",
				zap.Any("conn", c),
				zap.Error(err),
			)
			continue
		}

		// convert to our rpc message type and custom rpc format
		addr, err := addrOf(c)
		if err != nil {
			t.logger.Error("read",
				zap.Error(err))
			continue
		}
		rpc := RPC{
			From:    addr,
			Content: bytes.NewReader(buf.Bytes()),
		}

		t.recv <- rpc
	}

}

func (t *TcpTransport) Addr() net.Addr {
	return t.addr
}

func (t *TcpTransport) listen() {

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.logger.Error("accept error", zap.Error(err))
		}
		id, err := addrOf(conn)
		if err != nil {
			t.logger.Error("accept error", zap.Error(err))
		}
		t.connections[id] = conn
		go t.read(conn)

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
