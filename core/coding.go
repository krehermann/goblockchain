package core

import (
	"encoding/gob"
	"io"
)

type Encoder[T any] interface {
	Encode(T) error
}

type Decoder[T any] interface {
	Decode(T) error
}

type DefaultBlockEncoder struct {
	w io.Writer
}

func NewDefaultBlockEncoder(w io.Writer) *DefaultBlockEncoder {
	return &DefaultBlockEncoder{
		w: w,
	}
}

func (e DefaultBlockEncoder) Encode(b *Block) error {
	return gob.NewEncoder(e.w).Encode(b)
}

type DefaultBlockDecoder struct {
	r io.Reader
}

func NewDefaultBlockDecoder(r io.Reader) *DefaultBlockDecoder {
	return &DefaultBlockDecoder{
		r: r,
	}
}
func (d *DefaultBlockDecoder) Decode(b *Block) error {
	return gob.NewDecoder(d.r).Decode(b)
}

type GobTxEncoder struct {
	w io.Writer
}

func NewGobTxEncoder(w io.Writer) *GobTxEncoder {
	return &GobTxEncoder{
		w: w,
	}
}

func (e GobTxEncoder) Encode(t *Transaction) error {

	return gob.NewEncoder(e.w).Encode(t)
}

type GobTxDecoder struct {
	r io.Reader
}

func NewGobTxDecoder(r io.Reader) *GobTxDecoder {
	return &GobTxDecoder{
		r: r,
	}
}

func (d GobTxDecoder) Decode(t *Transaction) error {
	return gob.NewDecoder(d.r).Decode(t)
}
