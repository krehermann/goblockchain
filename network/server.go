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

type Peers struct {
}
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

func (s *Server) handleStatusMessageResponse(smsg *StatusMessageResponse) error {
	s.logger.Info("handleStatusMessageResponse",
		zap.Any("status", smsg),
	)

	return nil
}

func (s *Server) handleStatusMessageRequest(smsg *StatusMessageRequest) error {
	s.logger.Info("handleStatusMessageRequest",
		zap.Any("status", smsg),
	)

	// send my status to the requestor
	cur, err := s.currentStatus()
	if err != nil {
		return err
	}
	msg, err := newMessageFromStatusMessageResponse(cur)
	if err != nil {
		return err
	}

	return s.broadcast(msg)
}

func (s *Server) handleBlock(b *core.Block) error {
	s.logger.Info("handleBlock",
		zap.String("hash", b.Hash(core.DefaultBlockHasher{}).Prefix()),
	)
	err := s.chain.AddBlock(b)
	if err != nil {
		return err
	}
	go func() {
		err := s.broadcastBlock(b)
		if err != nil {
			s.errChan <- err
		}
	}()
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting server",
		zap.String("id", s.ID),
	)
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	s.initTransports()
	go s.handleRpcs(ctx)
	if s.isValidator {
		go s.handleValidtion(ctx)
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
			cancelFunc()
			break errorLoop
		}
	}

	return nil
}

func (s *Server) currentStatus() (*StatusMessageResponse, error) {
	status := new(StatusMessageResponse)

	hght := s.chain.Height()
	ver := uint32(0)
	header, err := s.chain.GetHeader(hght)
	if err == nil {
		ver = header.Version
	}
	status.CurrentHeight = hght
	status.Version = ver
	status.ServerID = s.ID
	return status, nil
}

func (s *Server) sendStartupStatusRequests() error {
	req := new(StatusMessageRequest)
	req.RequestorID = s.ID
	s.logger.Debug("sending startup status",
		zap.Any("msg", req),
	)

	msg, err := newMessageFromStatusMessageRequest(req)
	if err != nil {
		return err
	}
	s.broadcast(msg)
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
				s.errChan <- fmt.Errorf("server failed to decode rpc: %w", err)
				continue
			}
			err = s.ProcessMessage(decodedMsg)
			if err != nil {
				s.errChan <- fmt.Errorf("handle rpc: failed to process decoded message: %w", err)
			}
		case <-s.quitChan:
			s.logger.Info("handleRpcs received quit signal")
			break loop
		case <-ctx.Done():
			s.logger.Info("handleRpcs: received done")
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
			err := s.broadcastTx(tx)
			if err != nil {
				s.errChan <- err
			}
		}()
	}
	return err
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	s.logger.Debug("broadcastTx",
		zap.Any("hash", tx.Hash(&core.DefaultTxHasher{}).Prefix()))

	msg, err := newMessageFromTransaction(tx)
	if err != nil {
		return err
	}
	return s.broadcast(msg)
}

func (s *Server) broadcast(msg *Message) error {
	data, err := msg.Bytes()
	if err != nil {
		return err
	}
	if data == nil {
		panic("nil payload")
	}

	for _, trans := range s.Transports {
		if err := trans.Broadcast(CreatePayload(data)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) broadcastBlock(b *core.Block) error {
	s.logger.Info("broadcast block",
		zap.String("hash", b.Hash(core.DefaultBlockHasher{}).Prefix()),
	)
	msg, err := newMessageFromBlock(b)
	if err != nil {
		return err
	}
	return s.broadcast(msg)

}

func (s *Server) createNewBlock() error {
	s.logger.Info("creating block")

	currHeader, err := s.chain.GetHeader(s.chain.Height())
	if err != nil {
		return err
	}

	// TODO: add logic to determine how many txns can be in a block
	txns := s.mempool.Pending()

	block, err := core.NewBlockFromPrevHeader(currHeader, txns)
	if err != nil {
		return err
	}

	err = block.Sign(s.PrivateKey)
	if err != nil {
		return err
	}

	err = s.chain.AddBlock(block)
	if err != nil {
		return err
	}
	go func() {
		err := s.broadcastBlock(block)
		if err != nil {
			s.errChan <- err
		}
	}()
	// remember to clear our mempool after adding a block
	// would like to make this cleaner
	s.mempool.ClearPending()

	return nil
}

func (s *Server) initTransports() {
	// make each transport listen for messages
	for _, tr := range s.Transports {
		go func(tr Transport) {
			for rpc := range tr.Consume() {
				// we need to do something with the messages
				// we would like to keep it simple and flexible
				// to that end, we simply forward to a channel
				// owned by the server
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
