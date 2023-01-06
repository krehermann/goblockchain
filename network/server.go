package network

import (
	"context"
	"fmt"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"go.uber.org/zap"
)

var (
	defaultBlockTime = 5 * time.Second
)

type ServerOpts struct {
	// multiple transport layers
	Transports []Transport
	PrivateKey *crypto.PrivateKey
	BlockTime  time.Duration
	TxHasher   core.Hasher[*core.Transaction]
	Logger     *zap.Logger
	RPCHandler RPCHandler
}

type Server struct {
	ServerOpts
	// if server is elected as validator, server needs to know
	// when to consume mempool propose a block to the network
	// probably should be len(pool) >= maxTxPerBlock || t > blocktime
	blockTime   time.Duration
	mempool     *TxPool
	isValidator bool
	rpcChan     chan RPC
	quitChan    chan struct{}
	logger      *zap.Logger
}

func NewServer(opts ServerOpts) *Server {
	if opts.TxHasher == nil {
		opts.TxHasher = &core.DefaultTxHasher{}
	}
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewDevelopment()
	}
	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}
	s := &Server{
		ServerOpts:  opts,
		mempool:     NewTxPool(),
		blockTime:   opts.BlockTime,
		isValidator: (opts.PrivateKey != nil),
		rpcChan:     make(chan RPC),
		quitChan:    make(chan struct{}, 1),
		logger:      opts.Logger,
	}

	if s.ServerOpts.RPCHandler == nil {
		s.ServerOpts.RPCHandler = NewDefaultRPCHandler(s)
	}
	return s
}

func (s *Server) ProcessTransaction(from NetAddr, tx *core.Transaction) error {
	//	s.logger.Info("ProcessTransaction", zap.Any("tx", tx), zap.String("from", string(from)))
	return s.handleTransaction(tx)
}

func (s *Server) Start(ctx context.Context) error {

	errCh := make(chan error)
	ctx, cancelFunc := context.WithCancel(ctx)

	s.initTransports()
	go func() {
		blockTicker := time.NewTicker(s.blockTime)
	loop:
		for {
			select {
			case rpc := <-s.rpcChan:
				err := s.ServerOpts.RPCHandler.HandleRPC(rpc)
				if err != nil {
					errCh <- err
				}

			case <-s.quitChan:
				break loop

			// prevent busy loop
			case <-blockTicker.C:
				if s.isValidator {
					// need to call consensus logic here
					s.createNewBlock()
				}
			}
		}
	}()

errorLoop:
	for {
		select {
		case err := <-errCh:
			s.logger.Error(err.Error())
		case <-ctx.Done():
			cancelFunc()
			break errorLoop
		}
	}

	return nil
}

func (s *Server) handleTransaction(tx *core.Transaction) error {
	// 2 ways for txn to come in
	// 1. client, like a wallet creates a txn & sends the server,
	// server puts it into the mempool
	// 2. server broadcasts txn to connected nodes. this is really second
	// part of the first
	var err error

	err = tx.Verify()
	if err != nil {
		return err
	}
	tx.SetCreatedAt(time.Now().UTC())

	addedOk, err := s.mempool.Add(tx, s.ServerOpts.TxHasher)
	if err != nil {
		return err
	}

	if addedOk {
		s.logger.Info("added tx to mempool",
			zap.String("txn hash", tx.Hash(s.TxHasher).String()))
		// TODO broadcast to peers
	} else {
		s.logger.Info("skipped tx, already in mempool",
			zap.String("txn hash", tx.Hash(s.TxHasher).String()))

	}
	return nil
}

func (s *Server) createNewBlock() {
	fmt.Println("creating block")
}

func (s *Server) handleRPC(msg RPC) {
	fmt.Printf("dummy rpc handler: %+v\n", msg)
}

func (s *Server) initTransports() {
	// make each transport listen for messages
	for _, tr := range s.Transports {

		go func(tr Transport) {
			for msg := range tr.Consume() {
				// we need to do something with the messages
				// we would like to keep it simple and flexible
				// to that end, we simply forward to a channel
				// owned by the server
				s.rpcChan <- msg
			}
		}(tr)
	}
}
