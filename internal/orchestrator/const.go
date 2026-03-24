package orchestrator

import "time"

const (
	// WatchdogTimeout is the duration after which the watchdog will trigger if not reset
	WatchdogTimeout = 5 * time.Second
	// WatchdogInterval is the interval at which the watchdog will get pinged
	WatchdogInterval = 1 * time.Second
)
