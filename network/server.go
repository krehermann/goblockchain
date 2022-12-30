package network

import (
	"fmt"
	"time"
)

type ServerOpts struct {
	// multiple transport layers
	Transports []Transport
}

type Server struct {
	ServerOpts
	rpcChan  chan RPC
	quitChan chan struct{}
}

func NewServer(opts ServerOpts) *Server {
	return &Server{
		ServerOpts: opts,
		rpcChan:    make(chan RPC),
		quitChan:   make(chan struct{}, 1),
	}

}

func (s *Server) Start() error {
	s.initTransports()
	ticker := time.NewTicker(5 * time.Second)
loop:
	for {
		select {
		case msg := <-s.rpcChan:
			s.handleRPC(msg)

		case <-s.quitChan:
			break loop

		// prevent busy loop
		case <-ticker.C:
			fmt.Println("tick tock")
		}
	}
	return nil
}

func (s *Server) handleRPC(msg RPC) {
	fmt.Printf("dummy rpc handler: %+v\n", msg)
}

func (s *Server) initTransports() {
	// make each transport listen for messages
	for _, tr := range s.Transports {

		go func(tr Transport) {
			for msg := range tr.Consume() {
				// we need to do something with the messages
				// we would like to keep it simple and flexible
				// to that end, we simply forward to a channel
				// owned by the server
				s.rpcChan <- msg
			}
		}(tr)
	}
}
