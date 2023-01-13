package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/network"
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

	local := network.NewLocalTransport("local")
	peer0 := network.NewLocalTransport("peer0")
	peer1 := network.NewLocalTransport("peer1")
	peer2 := network.NewLocalTransport("peer2")

	fatalIfErr(cancelFunc, local.Connect(peer0))
	fatalIfErr(cancelFunc, peer0.Connect(local))

	fatalIfErr(cancelFunc, peer0.Connect(peer1))
	fatalIfErr(cancelFunc, peer1.Connect(peer2))

	initRemoteServers(ctx, peer0, peer1, peer2)

	go func() {
		cnt := 0
		for {
			fatalIfErr(cancelFunc, sendRandomTransaction(peer0, local))
			time.Sleep(1 * time.Second)
			cnt += 1
		}
	}()

	localServer := mustMakeServer(makeValidatorOpts("LOCAL", local))
	go localServer.Start(ctx)

	time.Sleep(10 * time.Second)
	cancelFunc()
	time.Sleep(2 * time.Second)
}

func initRemoteServers(ctx context.Context, trs ...network.Transport) {
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

func mustMakeServer(opts network.ServerOpts) *network.Server {
	s, err := network.NewServer(opts)
	fatalIfErr(nil, err)

	return s
}

func makeValidatorOpts(id string, tr network.Transport) network.ServerOpts {
	privKey := crypto.MustGeneratePrivateKey()
	return network.ServerOpts{
		PrivateKey: privKey,
		ID:         id,
		Transports: []network.Transport{tr},
	}
}

func makeNonValidatorOpts(id string, tr network.Transport) network.ServerOpts {
	return network.ServerOpts{
		ID:         id,
		Transports: []network.Transport{tr},
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
	msg := network.NewMessage(network.MessageTypeTx, txEncoded.Bytes())
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
		byte(a),
		byte(vm.InstructionPush),
		byte(b),
		byte(vm.InstructionPush),
		byte(vm.InstructionAdd),
	}

	return core.NewTransaction(txBytes)

}
