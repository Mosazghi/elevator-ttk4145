package orchestrator

import "time"

const (
	// the duration after which the watchdog will trigger if not reset
	watchdogTimeout = 5 * time.Second
	// the interval at which the watchdog will get pinged
	watchdogInterval = 1 * time.Second
)
