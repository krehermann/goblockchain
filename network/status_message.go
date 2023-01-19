package network

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
