package network

import (
	"context"
	"math/rand"
	"sync"

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

type Peers struct {
}
type ServerOpts struct {
	ID        string
	Transport Transport
	// multiple transport layers
	PeerTransports []Transport
	PrivateKey     crypto.PrivateKey
	BlockTime      time.Duration
	TxHasher       core.Hasher[*core.Transaction]
	Logger         *zap.Logger
	RPCDecodeFunc  RPCDecodeFunc
	RPCProcessor   RPCProcessor
	Blockchain     *core.Blockchain
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
		opts.RPCDecodeFunc = ExtractMessageFromRPC
	}
	if opts.ID == "" {
		opts.ID = strconv.FormatInt(int64(rand.Intn(1000)), 10)
	}
	opts.Logger = opts.Logger.Named(opts.ID)

	if opts.Blockchain == nil {
		genesis := createGenesis()
		chain, err := core.NewBlockchain(genesis, core.WithLogger(opts.Logger))
		if err != nil {
			return nil, err
		}
		opts.Blockchain = chain
	}
	s := &Server{
		ServerOpts:  opts,
		mempool:     NewTxPool(WithLogger(opts.Logger)),
		blockTime:   opts.BlockTime,
		isValidator: !opts.PrivateKey.IsZero(),
		rpcChan:     make(chan RPC),
		quitChan:    make(chan struct{}, 1),
		logger:      opts.Logger.Named("server"),
		errChan:     make(chan error, 1),
		chain:       opts.Blockchain,
	}

	// the server itself is the default rpc processor
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}
	return s, nil
}

func (s *Server) SetLogger(l *zap.Logger) {
	s.logger = l.Named(s.ID)
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting server",
		zap.String("id", s.ID),
	)

	wg := sync.WaitGroup{}
	s.initTransports()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.runRPCProcessor(ctx)
		s.logger.Sugar().Info("done handling rpcs")
	}()
	if s.isValidator {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runValidator(ctx)
			s.logger.Sugar().Info("done handling validation")
		}()
	}

	err := s.sendStartupStatusRequests()
	if err != nil {
		return err
	}

errorLoop:
	for {
		select {
		case err := <-s.errChan:
			if err != nil {
				s.logger.Error(err.Error())
			}
		case <-ctx.Done():
			s.logger.Info("received done")

			break errorLoop
		}
	}

	s.logger.Sugar().Info("waiting for shutdown of goroutines")
	wg.Wait()
	s.logger.Sugar().Info("shutdown. closing channels")
	/*
		close(s.rpcChan)
		s.rpcChan = nil
		close(s.errChan)
		s.errChan = nil
	*/
	return nil
}

// implement RPCProcessor interface
func (s *Server) ProcessMessage(dmsg *DecodedMessage) error {
	switch t := dmsg.Data.(type) {

	case *core.Transaction:
		s.logger.Info("ProcessMessage",
			zap.String("type", MessageTypeTx.String()),
			zap.String("from", string(dmsg.From)))

		return s.handleTransaction(t)
	case *core.Block:
		s.logger.Info("ProcessMessage",
			zap.String("type", MessageTypeBlock.String()),
			zap.String("from", string(dmsg.From)))
		return s.handleBlock(t)
	case *StatusMessageRequest:
		s.logger.Info("ProcessMessage",
			zap.String("type", MessageTypeStatusRequest.String()),
			zap.String("from", string(dmsg.From)))
		return s.handleStatusMessageRequest(t)

	case *StatusMessageResponse:
		s.logger.Info("ProcessMessage",
			zap.String("type", MessageTypeStatusResponse.String()),
			zap.String("from", string(dmsg.From)))
		return s.handleStatusMessageResponse(t)

	default:
		s.logger.Info("ProcessMessage",
			zap.Any("type", t),
			zap.String("from", string(dmsg.From)))

		return fmt.Errorf("invalid decoded message %v", t)
	}
}

// func (s *Server) Add
func (s *Server) initTransports() {
	// make each transport listen for messages
	for _, tr := range s.PeerTransports {
		go func(tr Transport) {
			s.logger.Debug("initializing consumer for peer",
				zap.String("peer address", string(tr.Addr())),
			)
			for rpc := range tr.Consume() {
				// we need to do something with the messages
				// we would like to keep it simple and flexible
				// to that end, we simply forward to a channel
				// owned by the server

				//hack to handle shutdown. need to think about the right way
				// to do this
				if s.rpcChan == nil {
					s.logger.Warn("rpc channel closed. aborting.")
					break
				}
				s.rpcChan <- rpc
			}
		}(tr)

	}
}

func createGenesis() *core.Block {
	h := &core.Header{
		Version:   1,
		DataHash:  types.Hash{},
		Height:    0,
		Timestamp: uint64(time.Unix(1000000, 0).Unix()),
	}
	return core.NewBlock(h, []*core.Transaction{})
}
