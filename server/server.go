package server

import (
	"context"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"fmt"
	"strconv"
	"time"

	"github.com/krehermann/goblockchain/api"
	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/network"
	"github.com/krehermann/goblockchain/protocol"
	"github.com/krehermann/goblockchain/types"
	"go.uber.org/zap"
)

var (
	defaultBlockTime = 5 * time.Second
)

type Peers struct {
	m     sync.RWMutex
	pipes map[string]network.Pipe
}

func NewPeers() *Peers {
	return &Peers{
		m:     sync.RWMutex{},
		pipes: make(map[string]network.Pipe),
	}
}

func (p *Peers) add(id string, pipe network.Pipe) {
	p.m.Lock()
	defer p.m.Unlock()
	p.pipes[id] = pipe
}

func (p *Peers) get(id string) (network.Pipe, bool) {
	p.m.RLock()
	defer p.m.RUnlock()
	pipe, exists := p.pipes[id]
	return pipe, exists
}

func (p *Peers) Slice() []network.Pipe {
	p.m.Lock()
	defer p.m.Unlock()
	var (
		keys []string
		out  []network.Pipe
	)
	for k := range p.pipes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pipe := p.pipes[k]
		out = append(out, pipe)
	}

	return out

}

func (p *Peers) Len() int {
	p.m.RLock()
	defer p.m.RUnlock()
	return len(p.pipes)
}
func (p *Peers) String() string {
	p.m.Lock()
	defer p.m.Unlock()
	var (
		keys []string
	)
	for k := range p.pipes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var temp []string
	for _, k := range keys {
		pipe := p.pipes[k]
		temp = append(temp, fmt.Sprintf("%s:%s", k, pipe.String()))
	}

	return strings.Join(temp, " , ")
}

type ServerOpts struct {
	ID              string
	ApiListenerAddr string
	Transport       network.Transport
	// multiple transport layers
	PrivateKey    crypto.PrivateKey
	BlockTime     time.Duration
	TxHasher      core.Hasher[*core.Transaction]
	Logger        *zap.Logger
	RPCDecodeFunc network.RPCDecodeFunc
	RPCProcessor  network.RPCProcessor
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
	rpcChan     chan network.RPC
	quitChan    chan struct{}
	logger      *zap.Logger
	errChan     chan error
	chain       *core.Blockchain
	Peers       *Peers
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
		mempool:     NewTxPool(WithLogger(opts.Logger), WithHasher(opts.TxHasher)),
		blockTime:   opts.BlockTime,
		isValidator: !opts.PrivateKey.IsZero(),
		rpcChan:     make(chan network.RPC),
		quitChan:    make(chan struct{}, 1),
		logger:      opts.Logger.Named("server"),
		errChan:     make(chan error, 1),
		chain:       opts.Blockchain,
		Peers:       NewPeers(),
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

func (s *Server) receive() {
	for rpc := range s.Transport.Recv() {
		s.rpcChan <- rpc
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting server",
		zap.String("id", s.ID),
	)

	//s.Connect(s.PeerTransports...)
	wg := sync.WaitGroup{}
	//	s.initTransports()
	// TODO: how to shut this down?
	go s.receive()

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

	if s.ApiListenerAddr != "" {
		apiServer, _ := api.NewServer(
			api.ServerConfig{
				ListenerAddr: s.ApiListenerAddr,

				Logger: s.logger.Named("api-server")},
			s.chain,
		)
		go apiServer.Start()
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
func (s *Server) ProcessMessage(dmsg *network.DecodedMessage) error {
	switch t := dmsg.Data.(type) {

	case *core.Transaction:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeTx.String()),
			zap.String("from", dmsg.From))

		return s.handleTransaction(t)
	case *core.Block:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeBlock.String()),
			zap.String("from", dmsg.From))
		return s.handleBlock(t)
	case *protocol.StatusMessageRequest:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeStatusRequest.String()),
			zap.String("from", dmsg.From))
		return s.handleStatusMessageRequest(t)

	case *protocol.StatusMessageResponse:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeStatusResponse.String()),
			zap.String("from", dmsg.From))
		return s.handleStatusMessageResponse(t)

	case *protocol.SubscribeMessageRequest:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeSubscribeRequest.String()),
			zap.String("from", dmsg.From))
		return s.handleSubscribeMessageRequest(t)

	case *protocol.SubscribeMessageResponse:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeSubscribeResponse.String()),
			zap.String("from", dmsg.From))
		return s.handleSubscribeMessageResponse(t)

	case *protocol.GetBlocksRequest:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeGetBlocksRequest.String()),
			zap.String("from", dmsg.From))
		return s.handleGetBlocksRequest(t)

	case *protocol.GetBlocksResponse:
		s.logger.Info("ProcessMessage",
			zap.String("type", protocol.MessageTypeGetBlocksResponse.String()),
			zap.String("from", dmsg.From))
		return s.handleGetBlocksResponse(t)

	default:
		s.logger.Info("ProcessMessage",
			zap.Any("type", t),
			zap.String("from", dmsg.From))

		return fmt.Errorf("invalid decoded message %v", t)
	}
}

// setup connection bi-directional connect
// in the transports in s and the inputs
func (s *Server) Connect(strangers ...network.Transport) error {
	for _, pt := range strangers {
		s.logger.Info("connect",
			zap.Any("from", s.Transport.Addr()),
			zap.Any("to", pt.Addr()))
		if s.Transport.Addr() == pt.Addr() {
			continue
		}
		// might need to handle multiple attempts to connect to same transport. thinking in transport itself, idempotent
		err := s.Transport.Connect(pt)
		if err != nil {
			return fmt.Errorf("error connecting from server %s to %s",
				s.Transport.Addr(), pt.Addr())
		}
	}
	return nil
}

// setup up by directional messages
// adds each other to peer list
func (s *Server) Join(peer *Server) error {
	s.logger.Info("join",
		zap.String("requestor", s.ID),
		zap.String("peer", peer.ID),
	)
	// request to peer
	err := s.Subscribe(peer.Transport)
	if err != nil {
		return err
	}

	err = peer.Subscribe(s.Transport)
	if err != nil {
		return err
	}
	// if ok, then
	return nil
}

// add s to leader's peers
// subscribe might be cleaner in the transport layer, analog to consume
func (s *Server) Subscribe(leader network.Transport) error {
	s.logger.Info("Subscribe",
		zap.String("to", leader.Addr().String()),
		zap.String("subscriber", s.Transport.Addr().String()),
	)

	pipe, exists := s.Transport.Get(leader.Addr().String())
	if !exists {
		s.logger.Error("subscribe don't have connection to leader")
	}

	req := &protocol.SubscribeMessageRequest{
		RequestorID: s.Transport.Addr().String(),
		Handle:      pipe.LocalAddr().String(), //s.ID,
	}

	msg, err := protocol.NewMessageFromSubscribeMessageRequest(req)
	if err != nil {
		return err
	}

	return s.send(leader.Addr(), msg)

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
