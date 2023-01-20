package network

import "github.com/krehermann/goblockchain/core"

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
	RequestorAddr NetAddr
}

type GetBlocksRequest struct {
	StartHeight uint32
}

type GetBlocksResponse struct {
	Blocks []*core.Block
}
