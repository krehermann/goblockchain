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

	// send length
	bs := make([]byte, 4)

	id := []byte(t.addr.String())
	msgLen := len(id) + 4 + len(payload.data) + 4

	// write entire message length
	binary.LittleEndian.PutUint32(bs, uint32(msgLen))
	_, err := peer.Write(bs)
	if err != nil {
		return err
	}

	// write length of id
	binary.LittleEndian.PutUint32(bs, uint32(len(id)))
	_, err = peer.Write(bs)
	if err != nil {
		return err
	}

	_, err = peer.Write(id)
	if err != nil {
		return err
	}

	// write length of id
	binary.LittleEndian.PutUint32(bs, uint32(len(payload.data)))
	_, err = peer.Write(bs)
	if err != nil {
		return err
	}

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

		_, err = io.ReadFull(r, buf[:int(length)])
		if err != nil {
			panic(err)
		}

		//	var buf bytes.Buffer
		//	b, err := ioutil.ReadAll(c)
		//_, err := io.Copy(&buf, c)
		if err != nil {
			t.logger.Error("reading from conn",
				zap.Any("conn", c),
				zap.Error(err),
			)
			continue
		}

		res := make([]byte, int(length))
		n := copy(res, buf[:int(length)])
		if n != int(length) {
			panic(fmt.Sprintf("copy failed want %d got %d", int(length), n))
		}

		// first 4 are len of id
		idLen := binary.LittleEndian.Uint32(res[:4])
		offset := uint32(4)
		id := string(res[offset : offset+idLen])
		offset += idLen
		// then payload len
		pLen := binary.LittleEndian.Uint32(res[offset : offset+4])
		offset += 4
		payload := (res[offset : offset+pLen])

		//io.CopyN(bufio.NewWriter( bytes.NewBuffer()res), bufio.NewReader(buf), int64(length))
		rpc := RPC{
			From:    id,
			Content: bytes.NewReader(payload),
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

/*
var delim = "\n"

func encoderSender(addr net.Addr) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString(addr.Network())
	buf.WriteString(delim)
	buf.WriteString(addr.String())
	buf.WriteString(delim)
	return buf.Bytes()
}

func decodeSender(b []byte) (net.Addr, error) {
	r := bufio.NewReader(bytes.NewReader(b))
	network,err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	str, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	switch network{
	case "tcp":
		return &net.
	}
}
*/
