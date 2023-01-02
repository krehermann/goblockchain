package core

type Storager interface {
	Put(b *Block) error
}

type MemStore struct {
}

func NewMemStore() *MemStore {
	return &MemStore{}
}

func (ms *MemStore) Put(b *Block) error {
	return nil
}
