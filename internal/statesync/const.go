package statesync

import "time"

const (
	// time it takes for a node to be considered lost
	NodeLostTimeout = time.Second * 3
	// time it takes for an PROCESSING order to be released due to timeout
	OrderProcessingTimeout = time.Second * 15
	// worldview broadcast interval
	BroadcastInterval = time.Millisecond * 50
	// time until the node can serverd again after its last order has timed out
	BlockNewOrderDuration = time.Second * 7
	//  time it takes for local node to consider itself disconnected from the network
	DisconnectedTimeout = BroadcastInterval * 2
	// ID for hall call order than has not been assigned to any node
	UnassignedID = -1
)
