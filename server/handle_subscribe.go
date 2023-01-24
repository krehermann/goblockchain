package server

import (
	"fmt"

	"github.com/krehermann/goblockchain/protocol"
	"go.uber.org/zap"
)

func (s *Server) handleSubscribeMessageRequest(smsg *protocol.SubscribeMessageRequest) error {
	s.logger.Info("handleSubscribeMessageRequest",
		zap.Any("status", smsg),
	)

	// add the requestor to my peers
	//exists := s.Transport.IsConnected(smsg.RequestorAddr)
	pipe, exists := s.Transport.Get(smsg.Handle)
	if !exists {
		return fmt.Errorf("subscription request from ghost connection %+v", smsg)
	}

	//s.Peers = append(s.Peers, pipe)
	s.Peers.add(smsg.RequestorID, pipe)
	s.logger.Debug("added subscriber", zap.String("id", smsg.RequestorID))
	msg := &protocol.SubscribeMessageResponse{
		ProviderId: s.Transport.Addr().String(),
	}
	resp, err := protocol.NewMessageFromSubscribeMessageResponse(msg)
	if err != nil {
		return err
	}

	return s.send(pipe.RemoteAddr(), resp)
}

func (s *Server) handleSubscribeMessageResponse(smsg *protocol.SubscribeMessageResponse) error {
	s.logger.Info("handleStatusMessageResponse",
		zap.Any("subscribed to", smsg),
	)

	pipe, exists := s.Transport.Get(smsg.ProviderId)
	if !exists {
		return fmt.Errorf("subscription response from ghost connection %+v", smsg)
	}

	//s.Peers = append(s.Peers, pipe)
	s.Peers.add(smsg.ProviderId, pipe)
	return nil
}
