package core

import (
	"encoding/gob"
	"io"
)

type Encoder[T any] interface {
	Encode(io.Writer, T) error
}

type Decoder[T any] interface {
	Decode(io.Reader, T) error
}

type DefaultBlockEncoder struct{}

func (dbe DefaultBlockEncoder) Encode(w io.Writer, b *Block) error {
	return gob.NewEncoder(w).Encode(b)
}

type DefaultBlockDecoder struct{}

func (dbe DefaultBlockDecoder) Decode(r io.Reader, b *Block) error {
	return gob.NewDecoder(r).Decode(b)
}
