package network

import (
	"context"
	"math/rand"

	"fmt"
	"strconv"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/types"
	"go.uber.org/zap"
)

var (
	defaultBlockTime = 5 * time.Second
)

type ServerOpts struct {
	ID string
	// multiple transport layers
	Transports    []Transport
	PrivateKey    crypto.PrivateKey
	BlockTime     time.Duration
	TxHasher      core.Hasher[*core.Transaction]
	Logger        *zap.Logger
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
	Blockchain    *core.Blockchain
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
	errChan     chan error
	chain       *core.Blockchain
}

func NewServer(opts ServerOpts) (*Server, error) {
	if opts.TxHasher == nil {
		opts.TxHasher = &core.DefaultTxHasher{}
	}
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewDevelopment()
	}
	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}
	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}
	if opts.ID == "" {
		opts.ID = strconv.FormatInt(int64(rand.Intn(1000)), 10)
	}
	if opts.Blockchain == nil {
		genesis := createGenesis()
		chain, err := core.NewBlockchain(genesis)
		if err != nil {
			return nil, err
		}
		opts.Blockchain = chain
	}
	opts.Logger = opts.Logger.Named(fmt.Sprintf("server-%s", opts.ID))
	s := &Server{
		ServerOpts:  opts,
		mempool:     NewTxPool(WithLogger(opts.Logger)),
		blockTime:   opts.BlockTime,
		isValidator: !opts.PrivateKey.IsZero(),
		rpcChan:     make(chan RPC),
		quitChan:    make(chan struct{}, 1),
		logger:      opts.Logger,
		errChan:     make(chan error, 1),
		chain:       opts.Blockchain,
	}

	// the server itself is the default rpc processor
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}
	return s, nil
}

// implement RPCProcessor interface
func (s *Server) ProcessMessage(dmsg *DecodedMessage) error {
	switch t := dmsg.Data.(type) {

	case *core.Transaction:
		s.logger.Info("ProcessMessage",
			zap.String("from", string(dmsg.From)),
			zap.String("type", MessageTypeTx.String()))

		return s.handleTransaction(t)

	default:
		return fmt.Errorf("invalid decoded message %v", t)
	}
}

func (s *Server) Start(ctx context.Context) error {

	ctx, cancelFunc := context.WithCancel(ctx)

	s.initTransports()
	go s.handleRpcs(ctx)
	if s.isValidator {
		go s.handleValidtion(ctx)
	}

errorLoop:
	for {
		select {
		case err := <-s.errChan:
			if err != nil {
				s.logger.Error(err.Error())
			}
		case <-ctx.Done():
			cancelFunc()
			break errorLoop
		}
	}

	return nil
}

func (s *Server) handleValidtion(ctx context.Context) {
	blockTicker := time.NewTicker(s.blockTime)
	s.logger.Info("starting validator loop", zap.String("block time", s.blockTime.String()))

validationLoop:
	for {

		select {
		// prevent busy loop
		case <-blockTicker.C:
			// need to call consensus logic here
			s.errChan <- s.createNewBlock()

		case <-ctx.Done():
			s.logger.Info("handleValidation recieved done")
			break validationLoop

		}
	}
}

func (s *Server) handleRpcs(ctx context.Context) {
	s.logger.Info("starting rpc handler loop")
loop:
	for {
		select {
		case rpc := <-s.rpcChan:
			decodedMsg, err := s.ServerOpts.RPCDecodeFunc(rpc)
			if err != nil {
				s.errChan <- err
			}
			err = s.ProcessMessage(decodedMsg)
			if err != nil {
				s.errChan <- err
			}
		case <-s.quitChan:
			s.logger.Info("handleRpcs received quit signal")
			break loop
		case <-ctx.Done():
			break loop
		}
	}

}

func (s *Server) handleTransaction(tx *core.Transaction) error {
	// 2 ways for txn to come in
	// 1. client, like a wallet creates a txn & sends the server,
	// server puts it into the mempool
	// 2. server broadcasts txn to connected nodes. this is really second
	// part of the first
	err := tx.Verify()
	if err != nil {
		return err
	}
	tx.SetCreatedAt(time.Now().UTC())

	isNewTx, err := s.mempool.Add(tx, s.ServerOpts.TxHasher)
	if err != nil {
		return err
	}

	if isNewTx {
		go func() {
			s.errChan <- s.broadcastTx(tx)
		}()
	}
	return err
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	msg, err := newMessageFromTransaction(tx)
	if err != nil {
		return err
	}
	return s.broadcast(msg)
}

func (s *Server) broadcast(msg *Message) error {
	payload, err := msg.Bytes()
	if err != nil {
		return err
	}

	for _, trans := range s.Transports {
		if err := trans.Broadcast(payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) createNewBlock() error {
	s.logger.Info("creating block")

	currHeader, err := s.chain.GetHeader(s.chain.Height())
	if err != nil {
		return err
	}

	block, err := core.NewBlockFromPrevHeader(currHeader, nil)
	if err != nil {
		return err
	}

	err = block.Sign(s.PrivateKey)
	if err != nil {
		return err
	}

	return s.chain.AddBlock(block)
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

func createGenesis() *core.Block {
	h := &core.Header{
		Version:   1,
		DataHash:  types.Hash{},
		Height:    0,
		Timestamp: uint64(time.Now().UTC().Unix()),
	}
	return core.NewBlock(h, []core.Transaction{})
}
