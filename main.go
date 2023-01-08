package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/network"
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
	peer1 := network.NewLocalTransport("peer1")
	peer2 := network.NewLocalTransport("peer2")
	peer3 := network.NewLocalTransport("peer3")

	fatalIfErr(local.Connect(peer1))
	fatalIfErr(peer1.Connect(local))

	fatalIfErr(peer1.Connect(peer2))
	fatalIfErr(peer2.Connect(peer3))

	initRemoteServers(ctx, peer1, peer2, peer3)

	go func() {
		cnt := 0
		for {
			fatalIfErr(sendRandomTransaction(peer1, local))
			time.Sleep(1 * time.Second)
			cnt += 1
		}
	}()

	localServer := mustMakeServer(makeValidatorOpts("LOCAL", local))
	go localServer.Start(context.Background())

	time.Sleep(10 * time.Second)
	cancelFunc()
}

func initRemoteServers(ctx context.Context, trs ...network.Transport) {
	zap.L().Info("initRemoteServers")

	for i, tr := range trs {
		s := mustMakeServer(makeNonValidatorOpts(
			fmt.Sprintf("remote-%d", i), tr))

		go func() {
			err := s.Start(ctx)
			fatalIfErr(err)
		}()
	}

}

func mustMakeServer(opts network.ServerOpts) *network.Server {
	s, err := network.NewServer(opts)
	fatalIfErr(err)

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
	zap.L().Info("sendTransaction",
		zap.String("from", string(from.Addr())),
		zap.String("to", string(to.Addr())))
	privKey := crypto.MustGeneratePrivateKey()
	data := []byte(strconv.FormatInt(rand.Int63(), 10))

	tx := core.NewTransaction(data)
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
	payload, err := msg.Bytes()
	if err != nil {
		return err
	}
	return from.SendMessage(to.Addr(), payload)

}

func fatalIfErr(err error) {
	if err != nil {
		zap.L().Fatal(err.Error())
	}
}
