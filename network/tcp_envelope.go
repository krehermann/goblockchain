package network

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"go.uber.org/zap"
)

// tcpEnvelope is wrapper for structured messages passed in the tcp transport
type tcpEnvelope struct {
	senderId string
	data     []byte
}

func newtcpEnvelope(id string, data []byte) *tcpEnvelope {
	return &tcpEnvelope{
		senderId: id,
		data:     data,
	}
}

func encodeEnvelope(e *tcpEnvelope) ([]byte, error) {
	var (
		logger = zap.L().Sugar()

		lenSize = 4 // size of uint32
		// output buffer
		buf = bytes.NewBuffer([]byte{})
		// buffer for length prefix
		lenPrefix = make([]byte, lenSize)

		id = []byte(e.senderId)
		// length of message to send
		msgLen = len(id) + lenSize + len(e.data) + lenSize
	)

	// write the message len, then for each field (len, data)
	binary.LittleEndian.PutUint32(lenPrefix, uint32(msgLen))
	n, err := buf.Write(lenPrefix)
	if err != nil {
		return nil, err
	}
	if n != lenSize {
		return nil, fmt.Errorf("incomplete write %d want %d", n, lenSize)
	}
	// encode  id
	binary.LittleEndian.PutUint32(lenPrefix, uint32(len(id)))
	_, err = buf.Write(lenPrefix)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(id)
	if err != nil {
		return nil, err
	}

	// encode payload
	binary.LittleEndian.PutUint32(lenPrefix, uint32(len(e.data)))
	n, err = buf.Write(lenPrefix)
	if err != nil {
		return nil, err
	}

	n, err = buf.Write(e.data)
	if err != nil {
		return nil, err
	}
	if n != len(e.data) {
		return nil, fmt.Errorf("corrupt write len %d != %d", n, len(e.data))
	}
	logger.Debugf("wrote data (%s), size %d", string(e.data), n)

	return buf.Bytes(), nil
}

func decodeEnvelope(b []byte, e *tcpEnvelope) error {

	// first 4 are len of id
	idLen := binary.LittleEndian.Uint32(b[:4])
	offset := uint32(4)
	id := string(b[offset : offset+idLen])
	offset += idLen
	// then payload len
	pLen := binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4
	payload := (b[offset : offset+pLen])

	e.senderId = id
	e.data = payload

	return nil
}
