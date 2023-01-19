package network

import (
	"bytes"
	"context"
	"math"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/vm"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetwork(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)

	blockTime := 100 * time.Millisecond

	l, err := zap.NewDevelopment()
	assert.NoError(t, err)
	/*
		observedZapCore, observedLogs := observer.New(zap.DebugLevel)
		observedLogger := zap.New(observedZapCore).Named("test")
	*/
	// hack
	zap.ReplaceGlobals(l)

	n := newNetwork(t, ctx, blockTime, l)

	v := generateValidator(t, l, "validator")
	nonValidators := generateNonValidators(t, l, []string{"r0"}...)

	n.register(v)
	n.register(nonValidators...)

	toplgy := new(topology)

	toplgy.connect(v.ID, nonValidators[0].ID)
	toplgy.connect(nonValidators[0].ID, v.ID)

	//toplgy.connect(nonValidators[0].ID, nonValidators[1].ID)
	//toplgy.connect(nonValidators[1].ID, nonValidators[2].ID)

	n.setTopology(toplgy)
	//n.connectAll()

	n.startServers()

	time.Sleep(4 * blockTime)

	lateComer := generateNonValidators(t, l, "late")[0]
	// the transport layer is kinda fucked
	// shenignans here
	n.register(lateComer)
	n.toplgy.connect(lateComer.ID, v.ID)
	n.initServer(lateComer, v)

	n.runFor(3, cancelFunc)

	servers := make([]*Server, 0)
	for _, server := range n.Servers {
		servers = append(servers, server)
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})

	for i := 0; i < len(servers); i++ {
		serverI := servers[i]
		hgtI := serverI.chain.Height()
		for j := i + 1; j < len(servers); j++ {
			serverJ := servers[j]
			hgtJ := serverJ.chain.Height()
			assert.InDelta(t, hgtI, hgtJ, 1,
				"heights differ too much between %s (%d) and %s (%d)",
				serverI.ID, hgtI,
				serverJ.ID, hgtJ,
			)
			common := int(math.Min(float64(hgtI), float64(hgtJ)))
			hdrI, err := serverI.chain.GetHeader(uint32(common))
			assert.NoError(t, err)
			hdrJ, err := serverJ.chain.GetHeader(uint32(common))
			assert.NoError(t, err)
			assert.True(t, reflect.DeepEqual(hdrI, hdrJ),
				"err comparing headers %s %+v: %s %+v",
				serverI.ID, hdrI,
				serverJ.ID, hgtJ)
		}
	}
}

func (n *network) runFor(nBlocks int, cancel context.CancelFunc) {
	x := 1.5 * float64(nBlocks)
	msec := float64(n.blockTime.Milliseconds())
	sleepMill := x * msec
	d := time.Duration(sleepMill) * time.Millisecond
	zap.L().Sugar().Infof("running network for %d milliseconds", d.Milliseconds())

	time.Sleep(d)
	zap.L().Sugar().Info("TEST CALLING CANCEL")
	cancel()
	zap.L().Sugar().Info("Waiting for server shutdown")
	n.wg.Wait()
	zap.L().Sugar().Info("NETWORK SHUTDOWN")
}

func (n *network) addPeer(from, to *Server) {
	zap.L().Info("networking adding peer",
		zap.String("from", from.ID),
		zap.String("to", to.ID),
	)

}

type topology struct {
	connections map[string][]string
}

func (t *topology) connect(from, to string) {
	if t.connections == nil {
		t.connections = make(map[string][]string)
	}
	conns, exist := t.connections[from]
	if !exist {
		conns = make([]string, 0)
	}
	conns = append(conns, to)
	t.connections[from] = conns
}

type network struct {
	t         *testing.T
	Servers   map[string]*Server
	ctx       context.Context
	logger    *zap.Logger
	toplgy    *topology
	blockTime time.Duration
	wg        sync.WaitGroup
}

func newNetwork(t *testing.T,
	ctx context.Context,
	blockTime time.Duration,
	logger *zap.Logger,
	servers ...*Server) *network {

	n := &network{
		t:         t,
		ctx:       ctx,
		logger:    logger.Named("network"),
		Servers:   make(map[string]*Server),
		blockTime: blockTime,
		wg:        sync.WaitGroup{},
	}
	n.register(servers...)
	return n
}

func (n *network) setTopology(t *topology) {
	n.toplgy = t
}

func (n *network) connectAll() {
	require.NotNil(n.t, n.toplgy, "network has no toplogy")
	require.NotNil(n.t, n.Servers, "network has no servers")

	for _, s := range n.Servers {
		n.initializeConnections(s)
	}

}

func (n *network) initializeConnections(s *Server) {
	fromServer, exists := n.Servers[s.ID]
	peerIds := n.toplgy.connections[s.ID]
	require.True(n.t, exists, "server id %s in topology but not in network", fromServer.ID)
	for _, peerId := range peerIds {
		peerServer, exists := n.Servers[peerId]
		require.True(n.t, exists, "server id %s in topology but not in network", peerId)

		require.NoError(n.t,
			fromServer.Connect(peerServer.Transport))

	}
}

func (n *network) initServer(s *Server, seeds ...*Server) {
	n.initializeConnections(s)
	// seed is the incoming connection to the given server
	for _, seed := range seeds {
		n.logger.Sugar().Info("seeding %s from %s", s.ID, seed.ID)

		require.NoError(n.t,
			s.Connect(seed.Transport))
	}

	n.t.Logf("test network starting %s", s.ID)
	n.wg.Add(1)
	go func(s *Server) {
		defer n.wg.Done()
		err := s.Start(n.ctx)
		n.logger.Sugar().Infof("server %s shutdown", s.ID)
		require.NoError(n.t, err)
	}(s)
}

func (n *network) startServers() {

	for _, server := range n.Servers {
		n.initServer(server)
	}
}

func (n *network) register(servers ...*Server) {
	n.logger.Sugar().Debugf("network register %d", len(servers))

	for _, s := range servers {
		require.NotEmpty(n.t, s.ID)
		// override the blockTime
		s.blockTime = n.blockTime
		//		s.SetLogger(n.observedLogger)
		n.Servers[s.ID] = s
	}
}

func generateValidator(t *testing.T, l *zap.Logger, id string) *Server {
	// TODO hook for non-local transport
	tr := NewLocalTransport(NetAddr(id))
	opts := makeValidatorOpts(id, tr)
	opts.Logger = l
	return mustMakeServer(t, opts)
}

func generateNonValidators(t *testing.T, l *zap.Logger, ids ...string) []*Server {

	sort.Strings(ids)
	out := make([]*Server, 0)

	for i, id := range ids {
		if i > 0 && id == ids[i-1] {
			continue
		}
		tr := NewLocalTransport(NetAddr(id))
		opts := makeNonValidatorOpts(id, tr)
		opts.Logger = l

		out = append(out, mustMakeServer(t, opts))

	}
	return out
}
func mustMakeServer(t *testing.T, opts ServerOpts) *Server {
	s, err := NewServer(opts)
	require.NoError(t, err)

	return s
}

func makeValidatorOpts(id string, tr Transport) ServerOpts {
	privKey := crypto.MustGeneratePrivateKey()
	return ServerOpts{
		PrivateKey: privKey,
		ID:         id,
		Transport:  tr,
		//PeerTransports: []Transport{tr},
	}
}

func makeNonValidatorOpts(id string, tr Transport) ServerOpts {
	return ServerOpts{
		ID:        id,
		Transport: tr,
		//PeerTransports: []Transport{tr},
	}
}

// helper for testing. remove later
func sendRandomTransaction(from, to Transport) error {
	privKey := crypto.MustGeneratePrivateKey()

	tx := transactionAdder()
	err := tx.Sign(privKey)
	if err != nil {
		return err
	}

	txEncoded := &bytes.Buffer{}
	err = tx.Encode(core.NewGobTxEncoder(txEncoded))
	if err != nil {
		return err
	}
	msg := NewMessage(MessageTypeTx, txEncoded.Bytes())
	d, err := msg.Bytes()
	if err != nil {
		return err
	}
	payload := CreatePayload(d)
	return from.Send(to.Addr(), payload)

}

func fatalIfErr(cancelFn context.CancelFunc, err error) {
	if err != nil {
		zap.L().Fatal(err.Error())
		if cancelFn != nil {
			cancelFn()
		}
	}
}

func transactionAdder() *core.Transaction {
	//	a := rand.Intn(8)
	//	b := rand.Intn(8)

	a := 5
	b := 2
	txBytes := []byte{
		// create bytes of key `abc`
		// make the xx key
		byte('a'),
		byte(vm.InstructionPushBytes),
		byte('b'),
		byte(vm.InstructionPushBytes),
		byte('c'),
		byte(vm.InstructionPushBytes),
		byte(3),
		byte(vm.InstructionPushInt),
		byte(vm.InstructionPack),

		// put two numbers
		byte(a),
		byte(vm.InstructionPushInt),
		byte(b),
		byte(vm.InstructionPushInt),
		// add,
		byte(vm.InstructionAddInt),
		// now the stack should be [abc], 7
		byte(vm.InstructionStore),
	}

	return core.NewTransaction(txBytes)

}
