package api

import (
	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/types"
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
}

type SubscribeMessageRequest struct {
	RequestorID   string
	RequestorAddr types.NetAddr
}

type GetBlocksRequest struct {
	RequestorID   string
	RequestorAddr types.NetAddr

	StartHeight uint32
}

type GetBlocksResponse struct {
	Blocks []*core.Block
}
