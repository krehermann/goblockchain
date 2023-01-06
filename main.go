package main

import (
	"bytes"
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/network"
)

// Server
// Transport => tcp, udp
// Block
// tx
// keypair

func main() {

	me := network.NewLocalTransport("local")
	peer := network.NewLocalTransport("remote")

	err := me.Connect(peer)
	if err != nil {
		log.Fatalf("%s", err)
	}
	err = peer.Connect(me)
	if err != nil {
		log.Fatalf("%s", err)
	}

	go func() {
		cnt := 0
		for {

			sendRandomTransaction(peer, me)
			time.Sleep(1 * time.Second)
			cnt += 1
		}
	}()

	opts := network.ServerOpts{
		Transports: []network.Transport{me, peer},
	}

	srv := network.NewServer(opts)
	srv.Start(context.Background())
}

// helper for testing. remove later
func sendRandomTransaction(from, to network.Transport) error {
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
