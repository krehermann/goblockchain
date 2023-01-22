package server

import (
	"fmt"

	"github.com/krehermann/goblockchain/api"
	"go.uber.org/zap"
)

func (s *Server) handleSubscribeMessageRequest(smsg *api.SubscribeMessageRequest) error {
	s.logger.Info("handleSubscribeMessageRequest",
		zap.Any("status", smsg),
	)

	// add the requestor to my peers
	exists := s.Transport.IsConnected(smsg.RequestorAddr)
	if !exists {
		return fmt.Errorf("subscription request from ghost connection %+v", smsg)
	}

	s.Peers = append(s.Peers, smsg.RequestorAddr)

	resp, err := api.NewMessageFromSubscribeMessageResponse(new(api.SubscribeMessageResponse))
	if err != nil {
		return err
	}

	return s.send(smsg.RequestorAddr, resp)
}

func (s *Server) handleSubscribeMessageResponse(smsg *api.SubscribeMessageResponse) error {
	s.logger.Info("handleStatusMessageResponse",
		zap.Any("subscribed to", smsg),
	)

	return nil
}
