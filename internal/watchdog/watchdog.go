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
	Timeout  chan bool
}

// Start new Watchdog timer with a given interval in seconds. Returns the WatchDog struct
func Start(duration float64) WatchDog {
	slog.Info("[Watchdog] Starting...")

	interval := time.Duration(duration * float64(time.Second))

	wd := WatchDog{
		pingChan: make(chan Emtpy, 1),
		stopChan: make(chan Emtpy),
		done:     make(chan Emtpy),
		Timeout:  make(chan bool, 1),
	}

	go func() {
		slog.Info("[Watchdog] Started")
		timer := time.NewTimer(interval)
		defer timer.Stop()
		defer close(wd.done)

		for {
			select {
			case <-timer.C:
				slog.Error("[Watchdog] Timed out")
				wd.Timeout <- true
				return

			case <-wd.pingChan:
				slog.Debug("[Watchdog] Ping received -> Timer is reset.")
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(interval)

			case <-wd.stopChan:
				slog.Info("[Watchdog] Timer Stopped")
				return
			}
		}
	}()

	return wd
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
