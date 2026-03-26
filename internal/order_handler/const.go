package order_handler

import "time"

const (
	// the duration between each order handler evaluation loop.
	pollInterval = 100 * time.Millisecond
)
