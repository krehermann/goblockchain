package main

import (
	"fmt"
	"log"
	"time"

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
			peer.SendMessage(me.Addr(), []byte(fmt.Sprintf("msg %d", cnt)))
			time.Sleep(1 * time.Second)
		}
	}()

	opts := network.ServerOpts{
		Transports: []network.Transport{me, peer},
	}
	print("hi")

	srv := network.NewServer(opts)
	srv.Start()
}
