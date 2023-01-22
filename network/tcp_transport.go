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
		zap.Any("from", t),
		zap.Any("to", o),
	)

	if t.IsConnected(o.Addr()) {
		return nil
	}
	conn, err := net.Dial(o.Addr().Network(), o.Addr().String())
	if err != nil {
		return err
	}

	t.lock.Lock()
	_, exists := t.connections[tcpPartner.Addr().String()]
	if !exists {
		t.connections[tcpPartner.Addr().String()] = conn
	}
	t.lock.Unlock()

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
	e := newtcpEnvelope(t.addr.String(), payload.data)
	buf, err := encodeEnvelope(e)
	if err != nil {
		return err
	}
	n, err := peer.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("corrupt write len %d != %d", n, len(payload.data))
	}
	t.logger.Sugar().Debugf("wrote %d to %s", n, peer.LocalAddr())
	return nil

}

func (t *TcpTransport) read(c net.Conn) {
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
		id, err := addrOf(conn)
		if err != nil {
			t.logger.Error("accept error", zap.Error(err))
		}
		t.lock.Lock()
		t.connections[id.String()] = conn
		t.lock.Unlock()
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
