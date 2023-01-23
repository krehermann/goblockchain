package network

import (
	"bufio"
	"bytes"
	"encoding/binary"
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
	connections map[string]net.Conn

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
		connections: make(map[string]net.Conn),
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

func (t *TcpTransport) Get(id string) (Pipe, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	addr, exists := t.connections[id]
	if !exists {
		return nil, false
	}
	return &pipeImpl{
		local:  addr.LocalAddr(),
		remote: addr.RemoteAddr()}, exists
}

func (t *TcpTransport) IsConnected(addr net.Addr) bool {
	t.lock.RLock()
	_, exists := t.connections[addr.String()]
	t.lock.RUnlock()
	//return peer, exists
	return exists
}
func (t *TcpTransport) Connect(o Transport) error {
	// has to be a tcp partner
	tcpPartner := o.(*TcpTransport)
	t.logger.Info("connect",
		zap.Any("from", t.addr.String()),
		zap.Any("to", o.Addr().String()),
	)

	if t.IsConnected(o.Addr()) {
		return nil
	}

	conn, err := net.Dial(o.Addr().Network(), o.Addr().String())
	if err != nil {
		return err
	}

	t.logger.Info("connect",
		zap.Any("conn.local", conn.LocalAddr().String()),
		zap.Any("conn.remote", conn.RemoteAddr().String()))

	t.lock.Lock()
	_, exists := t.connections[tcpPartner.Addr().String()]
	if !exists {
		t.connections[tcpPartner.Addr().String()] = conn
	}
	go t.read(conn)
	t.lock.Unlock()

	return nil
}

func (t *TcpTransport) Broadcast(payload Payload) error {
	var conns []net.Conn
	t.lock.RLock()
	for _, conn := range t.connections {
		conns = append(conns, conn)
	}
	t.lock.RUnlock()

	for _, conn := range conns {
		err := t.send(conn, payload)
		if err != nil {
			t.logger.Error("tcp broadcast. error sending",
				zap.Error(err))
		}
	}
	return nil
}

func (t *TcpTransport) Send(to net.Addr, payload Payload) error {
	t.logger.Debug("send message",
		zap.String("from", string(t.addr.String())),
		zap.String("to", string(to.String())),
		zap.Int("msg length", len(payload.data)),
	)

	t.lock.RLock()
	peer, exists := t.connections[to.String()]
	t.lock.RUnlock()

	if !exists {
		return fmt.Errorf("%s unknown peer %s", t.addr, to)
	}

	return t.send(peer, payload)
}
func (t *TcpTransport) send(conn net.Conn, payload Payload) error {

	e := newtcpEnvelope(t.addr.String(), payload.data)
	buf, err := encodeEnvelope(e)
	if err != nil {
		return err
	}
	n, err := conn.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("corrupt write len %d != %d", n, len(payload.data))
	}
	t.logger.Sugar().Debugf("wrote %d to %s", n, conn.LocalAddr())
	return nil

}

func (t *TcpTransport) read(c net.Conn) {
	t.logger.Info("reading connection",
		zap.Any("conn.local", c.LocalAddr().String()),
		zap.Any("conn.remote", c.RemoteAddr().String()))

	buf := make([]byte, 64*1024)
	lenBuf := make([]byte, 4)

	r := bufio.NewReader(c)
	for {

		// read  to get the length
		_, err := r.Read(lenBuf)
		if err != nil {
			panic(err)
		}
		length := binary.LittleEndian.Uint32(lenBuf)
		t.logger.Sugar().Debugf("read msg len %d", length)

		_, err = io.ReadFull(r, buf[:int(length)])
		if err != nil {
			panic(err)
		}

		var envelope tcpEnvelope
		err = decodeEnvelope(buf[:int(length)], &envelope)
		if err != nil {
			panic(err)
		}

		rpc := RPC{
			From:    envelope.senderId,
			Content: bytes.NewReader(envelope.data),
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
		/*
			id, err := conn.RemoteAddr() //addrOf(conn)
			if err != nil {
				t.logger.Error("accept error", zap.Error(err))
			}
		*/
		id := conn.RemoteAddr()
		t.logger.Debug("accepted connection",
			zap.Any("conn", conn),
			zap.Any("conn.remote", conn.RemoteAddr().String()),
			zap.Any("conn.local", conn.LocalAddr().String()),

			zap.String("id", id.String()),
		)

		t.lock.Lock()
		t.connections[id.String()] = conn
		t.lock.Unlock()
		go t.read(conn)

	}
}

func (t *TcpTransport) GetConn(to net.Addr) (net.Conn, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	c, exists := t.connections[to.String()]
	return c, exists
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
