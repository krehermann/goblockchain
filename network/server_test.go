package network

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/vm"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetwork(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	n := newNetwork(t, ctx)

	l, err := zap.NewDevelopment()
	assert.NoError(t, err)
	// hack
	zap.ReplaceGlobals(l)
	//	l = n.observedLogger
	v := generateValidator(t, l, "validator")
	nonValidators := generateNonValidators(t, l, []string{"r0", "r1", "r2"}...)

	n.register(v)
	n.register(nonValidators...)

	toplgy := new(topology)

	toplgy.connect(v.ID, nonValidators[0].ID)
	toplgy.connect(nonValidators[0].ID, v.ID)
	toplgy.connect(nonValidators[0].ID, nonValidators[1].ID)
	toplgy.connect(nonValidators[1].ID, nonValidators[2].ID)

	n.setTopology(toplgy)
	n.initializeConnections()

	go n.startServers()

	runFor(13*time.Second, cancelFunc)
	// let network come to steady state
	time.Sleep(5 * time.Second)

	hdrMap := make(map[string]*core.Header)
	for id, server := range n.Servers {
		hdr, err := server.chain.GetHeader(server.chain.Height())
		assert.NoError(t, err)
		hdrMap[id] = hdr
	}

	for id, hdr := range hdrMap {
		for otherId, otherHdr := range hdrMap {
			if id == otherId {
				continue
			}
			assert.Equal(t, hdr.Height, otherHdr.Height, "height mismatch between %s:%d != %s:%d",
				id, hdr.Height,
				otherId, otherHdr.Height)
			assert.True(t, reflect.DeepEqual(hdr, otherHdr),
				"err comparing headers %s %+v: %s %+v",
				id, hdr,
				otherId, otherHdr)
		}
	}

}

func runFor(d time.Duration, cancel context.CancelFunc) {
	time.Sleep(d)
	cancel()
}

func (n *network) addPeer(from, to *Server) {
	zap.L().Info("networking adding peer",
		zap.String("from", from.ID),
		zap.String("to", to.ID),
	)
	fromTransport := from.Transports[0]
	require.NotEmpty(n.t, fromTransport)

	toTransport := to.Transports[0]
	require.NotEmpty(n.t, toTransport)

	require.NoError(n.t,
		fromTransport.Connect(toTransport))
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
	t              *testing.T
	Servers        map[string]*Server
	ctx            context.Context
	observedLogs   *observer.ObservedLogs
	observedLogger *zap.Logger
	toplgy         *topology
	// cancelFunc?
}

func newNetwork(t *testing.T, ctx context.Context, servers ...*Server) *network {

	observedZapCore, observedLogs := observer.New(zap.DebugLevel)
	observedLogger := zap.New(observedZapCore).Named("network")

	// override the logger
	n := &network{
		t:              t,
		ctx:            context.Background(),
		observedLogs:   observedLogs,
		observedLogger: observedLogger,
		Servers:        make(map[string]*Server),
	}
	n.register(servers...)
	return n
}

func (n *network) setTopology(t *topology) {
	n.toplgy = t
}

func (n *network) initializeConnections() {
	require.NotNil(n.t, n.toplgy, "network has no toplogy")
	require.NotNil(n.t, n.Servers, "network has no servers")

	for fromId, peerIds := range n.toplgy.connections {
		fromServer, exists := n.Servers[fromId]
		require.True(n.t, exists, "server id %s in topology but not in network", fromId)
		for _, peerId := range peerIds {
			peerServer, exists := n.Servers[peerId]
			require.True(n.t, exists, "server id %s in topology but not in network", peerId)

			n.addPeer(fromServer, peerServer)
		}
	}
}

func (n *network) startServers() {
	for id, server := range n.Servers {
		n.t.Logf("test network starting %s", id)
		go func(s *Server) {

			err := s.Start(n.ctx)
			require.NoError(n.t, err)

		}(server)
	}
}

func (n *network) register(servers ...*Server) {
	n.observedLogger.Sugar().Debugf("network register %d", len(servers))

	for _, s := range servers {
		require.NotEmpty(n.t, s.ID)
		// override the logger
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

/*
func initRemoteServers(ctx context.Context, trs ...Transport) {
	zap.L().Info("initRemoteServers")

	ctx, cancelFunc := context.WithCancel(ctx)
	for i, tr := range trs {
		s := mustMakeServer(makeNonValidatorOpts(
			fmt.Sprintf("remote-%d", i), tr))

		go func() {
			err := s.Start(ctx)
			fatalIfErr(cancelFunc, err)
		}()
	}

}
*/
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
		Transports: []Transport{tr},
	}
}

func makeNonValidatorOpts(id string, tr Transport) ServerOpts {
	return ServerOpts{
		ID:         id,
		Transports: []Transport{tr},
	}
}

// helper for testing. remove later
func sendRandomTransaction(from, to Transport) error {
	privKey := crypto.MustGeneratePrivateKey()

	/*
		data := []byte(strconv.FormatInt(rand.Int63(), 10))

		tx := core.NewTransaction(data)
	*/
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
