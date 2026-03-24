package statesync

import "time"

const (
	NodeLostTimeout        = time.Second * 3       // time it takes for a node to be considered lost
	OrderProcessingTimeout = time.Second * 15      // time it takes for an PROCESSING order to be released due to timeout
	BroadcastInterval      = time.Millisecond * 50 // worldview broadcast interval
	BlockNewOrderDuration     = time.Second * 20
)
