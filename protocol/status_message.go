package protocol

import (
	"github.com/krehermann/goblockchain/core"
)

type StatusMessageResponse struct {
	ServerID      string
	CurrentHeight uint32
	Version       uint32
}

type StatusMessageRequest struct {
	RequestorID string
}

type SubscribeMessageResponse struct {
	ProviderId string
}

type SubscribeMessageRequest struct {
	RequestorID string
	Handle      string
	//RequestorAddr net.Addr
}

type GetBlocksRequest struct {
	RequestorID string
	//RequestorAddr net.Addr

	StartHeight uint32
}

type GetBlocksResponse struct {
	Blocks []*core.Block
}
