package statesync

import "time"

const (
	// time it takes for a node to be considered lost
	nodeLostTimeout = time.Second * 3
	// time it takes for an PROCESSING order to be released due to timeout
	orderProcessingTimeout = time.Second * 15
	// worldview broadcast interval
	broadcastInterval = time.Millisecond * 50
	// time until the node can serverd again after its last order has timed out
	blockNewOrderDuration = time.Second * 7
	//  time it takes for local node to consider itself disconnected from the network
	disconnectedTimeout = broadcastInterval * 2
)
