package statesync

import "time"

const (
	NodeLostTimeout        = time.Second * 3       // time it takes for a node to be considered lost
	OrderProcessingTimeout = time.Second * 20      // time it takes for an PROCESSING order to be released due to timeout
	BroadcastInterval      = time.Millisecond * 50 // worldview broadcast interval
	BlockNewOrderDuration  = time.Second * 5
	DisconnectedTimeout    = BroadcastInterval * 2 //  time it takes for local node to consider itself disconnected from the network
	UnassignedID           = -1                    // ID for hall call order than has not been assigned to any node
)
