package order_handler

import "time"

const (
	// pollInterval is the duration between each order handler evaluation loop.
	pollInterval = 100 * time.Millisecond
)
