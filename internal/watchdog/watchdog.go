package watchdog

import (
	"log/slog"
	"time"

	. "github.com/Mosazghi/elevator-ttk4145/shared"
)

type WatchDog struct {
	pingChan chan Empty
	stopChan chan Empty
	done     chan Empty
	Timeout  chan Empty
	duration time.Duration
}

func New(duration time.Duration) *WatchDog {
	return &WatchDog{
		pingChan: make(chan Empty, 1),
		stopChan: make(chan Empty),
		done:     make(chan Empty),
		Timeout:  make(chan Empty, 1),
		duration: duration,
	}
}

// Start new Watchdog timer with a given interval in seconds. Returns the WatchDog struct
func (wd *WatchDog) Start() {
	slog.Info("[Watchdog] Starting...")

	slog.Info("[Watchdog] Started")
	timer := time.NewTimer(wd.duration)
	defer timer.Stop()
	defer close(wd.done)

	for {
		select {
		case <-timer.C:
			slog.Error("[Watchdog] Timed out")
			wd.Timeout <- Empty{}
			return

		case <-wd.pingChan:
			slog.Debug("[Watchdog] Ping received -> Timer is reset.")
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(wd.duration)

		case <-wd.stopChan:
			slog.Info("[Watchdog] Timer Stopped")
			return
		}
	}
}

// Stop stops the watchdog timer
func (wd *WatchDog) Stop() {
	select {
	case <-wd.done:
		return
	default:
		close(wd.stopChan)
		<-wd.done
	}
}

// Ping resets the watchdog timer, postponing the timeout
func (wd *WatchDog) Ping() {
	wd.pingChan <- Empty{}
}
