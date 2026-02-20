package statesync

import "time"

const (
	NodeLostTimeout        = time.Second * 3        // time it takes for a node to be considered lost
	OrderProcessingTimeout = time.Second * 8        // time it takes for an PROCESSING order to be released due to timeout
	BroadcastInterval      = time.Millisecond * 800 // worldview broadcast interval
)
