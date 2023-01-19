package network

import (
	"context"
	"fmt"
	"time"

	"github.com/krehermann/goblockchain/core"
	"go.uber.org/zap"
)

func (s *Server) runValidator(ctx context.Context) {
	blockTicker := time.NewTicker(s.blockTime)
	s.logger.Info("starting validator loop", zap.String("block time", s.blockTime.String()))

validationLoop:
	for {

		select {
		// prevent busy loop
		case <-blockTicker.C:
			// need to call consensus logic here
			err := s.createNewBlock()
			if err != nil {
				s.errChan <- err
			}
		case <-ctx.Done():
			s.logger.Info("handleValidation recieved done")
			break validationLoop

		}
	}
}

func (s *Server) runRPCProcessor(ctx context.Context) {
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

	for _, trans := range s.PeerTransports {
		if err := trans.Broadcast(CreatePayload(data)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) send(addr NetAddr, msg *Message) error {
	data, err := msg.Bytes()
	if err != nil {
		return err
	}
	if data == nil {
		panic("nil payload")
	}
	return s.Transport.Send(addr, CreatePayload(data))
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
