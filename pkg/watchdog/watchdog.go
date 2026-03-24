// Package watchdog provides a simple timer mechanism to detect and handle timeouts in various components of the elevator system.
package watchdog

import (
	"log/slog"
	"time"

	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
)

// WatchDog implements a simple watchdog timer.
type WatchDog struct {
	pingChan chan Empty
	stopChan chan Empty
	done     chan Empty
	Timeout  chan Empty
	duration time.Duration
}

// New creates a watchdog with the given timeout duration.
func New(duration time.Duration) *WatchDog {
	return &WatchDog{
		pingChan: make(chan Empty, 1),
		stopChan: make(chan Empty),
		done:     make(chan Empty),
		Timeout:  make(chan Empty, 1),
		duration: duration,
	}
}

// Start runs the timer loop until timeout or explicit stop.
func (wd *WatchDog) Start() {
	timer := time.NewTimer(wd.duration)
	defer timer.Stop()
	defer close(wd.done)

	for {
		select {
		case <-timer.C:
			slog.Warn("timed out")
			wd.Timeout <- Empty{}
			return

		case <-wd.pingChan:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(wd.duration)

		case <-wd.stopChan:
			slog.Info("timer Stopped")
			return
		}
	}
}

// Stop terminates the timer loop and waits for shutdown completion.
func (wd *WatchDog) Stop() {
	select {
	case <-wd.done:
		return
	default:
		close(wd.stopChan)
		<-wd.done
	}
}

// Ping requests a timer reset; extra pings are coalesced.
func (wd *WatchDog) Ping() {
	select {
	case wd.pingChan <- Empty{}:
	default:
	}
}
