package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/network"
	"github.com/krehermann/goblockchain/protocol"
	"go.uber.org/zap"
)

func (s *Server) runValidator(ctx context.Context) {
	blockTicker := time.NewTicker(s.blockTime)
	s.logger.Info("starting validator loop", zap.String("block time", s.blockTime.String()))

validationLoop:
	for {

		select {
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

func (s *Server) currentStatus() (*protocol.StatusMessageResponse, error) {
	status := new(protocol.StatusMessageResponse)

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
	req := new(protocol.StatusMessageRequest)
	req.RequestorID = s.ID
	s.logger.Debug("sending startup status",
		zap.Any("msg", req),
	)

	msg, err := protocol.NewMessageFromStatusMessageRequest(req)
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

	isNewTx, err := s.mempool.Add(tx)
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
		zap.Any("hash", tx.Hash().Prefix()))

	msg, err := protocol.NewMessageFromTransaction(tx)
	if err != nil {
		return err
	}
	return s.broadcast(msg)
}

func (s *Server) broadcast(msg *protocol.Message) error {
	if s.Peers.Len() == 0 {
		s.logger.Warn("broadcasting without peers")
	}
	data, err := msg.Bytes()
	if err != nil {
		return err
	}
	return s.Transport.Broadcast(network.CreatePayload(data))
}

func (s *Server) send(addr net.Addr, msg *protocol.Message) error {
	data, err := msg.Bytes()
	if err != nil {
		return err
	}
	if data == nil {
		panic("nil payload")
	}
	return s.Transport.Send(addr, network.CreatePayload(data))
}

func (s *Server) broadcastBlock(b *core.Block) error {
	s.logger.Info("broadcast block",
		zap.String("hash", b.Hash(core.DefaultBlockHasher{}).Prefix()),
	)
	msg, err := protocol.NewMessageFromBlock(b)
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

func (s *Server) handleStatusMessageResponse(smsg *protocol.StatusMessageResponse) error {
	s.logger.Info("handleStatusMessageResponse",
		zap.Any("status", smsg),
	)

	return nil
}

func (s *Server) handleStatusMessageRequest(smsg *protocol.StatusMessageRequest) error {
	s.logger.Info("handleStatusMessageRequest",
		zap.Any("status", smsg),
	)

	// send my status to the requestor
	cur, err := s.currentStatus()
	if err != nil {
		return err
	}
	msg, err := protocol.NewMessageFromStatusMessageResponse(cur)
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
		s.logger.Error("what kind of error is this",
			zap.Error(err))
		syncErr := new(core.ErrOutOfSync)
		if errors.As(err, &syncErr) {
			s.logger.Error("got out of sync error", zap.Error(err))
			//if syncErr.Lag >  {
			// self heal
			s.requestBlocks()
			//}
		} else {
			return err
		}
	} else {
		go func() {
			err := s.broadcastBlock(b)
			if err != nil {
				s.errChan <- err
			}
		}()
	}
	return nil
}

func (s *Server) requestBlocks() error {
	n := s.chain.Height() + 1
	req := &protocol.GetBlocksRequest{
		RequestorID: s.Transport.Addr().String(),
		StartHeight: n,
	}
	s.logger.Info("requesting blocks",
		zap.Any("request", req),
	)

	msg, err := protocol.NewMessageFromGetBlocksRequest(req)

	if err != nil {
		return err
	}

	return s.broadcast(msg)

}

func (s *Server) handleGetBlocksRequest(msg *protocol.GetBlocksRequest) error {
	s.logger.Info("handleGetBlocksRequest",
		zap.Any("req", msg),
	)

	pipe, exists := s.Peers.get(msg.RequestorID)
	if !exists {
		s.logger.Error("handleGetBlocks unknown requestor",
			zap.String("id", msg.RequestorID),
		)
		return fmt.Errorf("cannot service request to unknown requestor %s", msg.RequestorID)
	}

	blocksToSend := make([]*core.Block, 0)
	for i := msg.StartHeight; i <= s.chain.Height(); i++ {
		block, err := s.chain.GetBlockAt(i)
		if err != nil {
			return err
		}
		blocksToSend = append(blocksToSend, block)
	}

	resp := &protocol.GetBlocksResponse{
		Blocks: blocksToSend,
	}
	out, err := protocol.NewMessageFromGetBlocksResponse(resp)
	if err != nil {
		return err
	}

	s.send(pipe.RemoteAddr(), out)
	return nil
}

func (s *Server) handleGetBlocksResponse(msg *protocol.GetBlocksResponse) error {
	s.logger.Info("handleGetBlocksResponse",
		zap.Any("response", msg),
	)

	for _, b := range msg.Blocks {
		if !s.chain.HasBlockAtHeight(b.Height) {
			err := s.chain.AddBlock(b)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
