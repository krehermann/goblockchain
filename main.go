package main

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/krehermann/goblockchain/api"
	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/network"
	"github.com/krehermann/goblockchain/server"
	"github.com/krehermann/goblockchain/vm"
	"go.uber.org/zap"
)

// Server
// Transport => tcp, udp
// Block
// tx
// keypair

func main() {

	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)

	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)

	local := network.NewLocalTransport(network.LocalAddr("local"))
	peer0 := network.NewLocalTransport(network.LocalAddr("peer0"))
	// peer1 := network.NewLocalTransport("peer1")
	// peer2 := network.NewLocalTransport("peer2")
	remotes := initRemoteServers(ctx, peer0)

	go func() {
		cnt := 0
		for {
			//			fatalIfErr(cancelFunc, sendRandomTransaction(peer0, local))
			time.Sleep(1 * time.Second)
			cnt += 1
		}
	}()

	localServer := mustMakeServer(makeValidatorOpts("LOCAL", local))
	//	localServer.PeerTransports = append(localServer.PeerTransports, remotes[0].Transport)

	go localServer.Start(ctx)
	err = remotes[0].Connect(localServer.Transport)
	fatalIfErr(cancelFunc, err)
	err = remotes[0].Subscribe(localServer.Transport)
	fatalIfErr(cancelFunc, err)

	time.Sleep(8 * time.Second)
	cancelFunc()
	time.Sleep(2 * time.Second)

	lhdr, err := localServer.Blockchain.GetHeader(localServer.Blockchain.Height())
	if err != nil {
		zap.L().Sugar().Fatalf("err getting local header %s", err)
	}
	for _, remote := range remotes {
		if localServer.Blockchain.Height() != remote.Blockchain.Height() {
			zap.L().Sugar().Errorf(
				"local height (%d) != remote height[%s](%d)",
				localServer.Blockchain.Height(),
				remote.ID,
				remote.Blockchain.Height(),
			)
			rhdr, err := remote.Blockchain.GetHeader(remote.Blockchain.Height())
			if err != nil {
				zap.L().Sugar().Fatalf("err getting remote header %s %s", remote.ID, err)
			}
			if !reflect.DeepEqual(lhdr, rhdr) {
				zap.L().Error("err comparing header",
					zap.Any("local", lhdr),
					zap.Any(remote.ID, rhdr))
			} else {
				zap.L().Sugar().Infof("local = %s", remote.ID)
			}
		}
	}
}

func initRemoteServers(ctx context.Context, trs ...network.Transport) []*server.Server {
	zap.L().Info("initRemoteServers")
	out := make([]*server.Server, 0)
	ctx, cancelFunc := context.WithCancel(ctx)
	for i, tr := range trs {
		s := mustMakeServer(makeNonValidatorOpts(
			fmt.Sprintf("remote-%d", i), tr))
		out = append(out, s)
		go func() {
			err := s.Start(ctx)
			fatalIfErr(cancelFunc, err)
		}()
	}
	return out
}

func mustMakeServer(opts server.ServerOpts) *server.Server {
	s, err := server.NewServer(opts)
	fatalIfErr(nil, err)

	return s
}

func makeValidatorOpts(id string, tr network.Transport) server.ServerOpts {
	privKey := crypto.MustGeneratePrivateKey()
	return server.ServerOpts{
		PrivateKey: privKey,
		ID:         id,
		Transport:  tr,
	}
}

func makeNonValidatorOpts(id string, tr network.Transport) server.ServerOpts {
	return server.ServerOpts{
		ID:        id,
		Transport: tr,
	}
}

// helper for testing. remove later
func sendRandomTransaction(from, to network.Transport) error {
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
	msg := api.NewMessage(api.MessageTypeTx, txEncoded.Bytes())
	d, err := msg.Bytes()
	if err != nil {
		return err
	}
	payload := network.CreatePayload(d)
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
