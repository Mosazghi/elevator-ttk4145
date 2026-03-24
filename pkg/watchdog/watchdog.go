// Package watchdog provides a simple timer mechanism to detect and handle timeouts in various components of the elevator system.
package watchdog

import (
	"time"

	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
)

// WatchDog implements a simple watchdog timer.
type WatchDog struct {
	pingChan    chan Empty
	stopChan    chan Empty
	doneChan    chan Empty
	TimeoutChan chan Empty
	duration    time.Duration
}

// New creates a watchdog with the given timeout duration.
func New(duration time.Duration) *WatchDog {
	return &WatchDog{
		pingChan:    make(chan Empty, 1),
		stopChan:    make(chan Empty),
		doneChan:    make(chan Empty),
		TimeoutChan: make(chan Empty, 1),
		duration:    duration,
	}
}

// Start runs the timer loop until timeout or explicit stop.
func (watchdog *WatchDog) Start() {
	timer := time.NewTimer(watchdog.duration)
	defer timer.Stop()
	defer close(watchdog.doneChan)

	for {
		select {
		case <-timer.C:
			watchdog.TimeoutChan <- Empty{}
			return

		case <-watchdog.pingChan:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(watchdog.duration)

		case <-watchdog.stopChan:
			return
		}
	}
}

// Stop terminates the timer loop and waits for shutdown completion.
func (watchdog *WatchDog) Stop() {
	select {
	case <-watchdog.doneChan:
		return
	default:
		close(watchdog.stopChan)
		<-watchdog.doneChan
	}
}

// Ping requests a timer reset.
// Uses non-blocking send to avoid deadlocks if the watchdog is already stopped or timed out.
func (watchdog *WatchDog) Ping() {
	select {
	case watchdog.pingChan <- Empty{}:
	default:
	}
}
