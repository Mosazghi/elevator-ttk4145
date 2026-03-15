package watchdog

import (
	"log/slog"
	"time"

	. "github.com/Mosazghi/elevator-ttk4145/shared"
)

type WatchDog struct {
	pingChan chan Emtpy
	stopChan chan Emtpy
	done     chan Emtpy
	Timeout  chan Emtpy
	duration time.Duration
}

func New(duration time.Duration) *WatchDog {
	return &WatchDog{
		pingChan: make(chan Emtpy, 1),
		stopChan: make(chan Emtpy),
		done:     make(chan Emtpy),
		Timeout:  make(chan Emtpy, 1),
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
			wd.Timeout <- Emtpy{}
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
	wd.pingChan <- Emtpy{}
}
