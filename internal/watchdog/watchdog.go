package watchdog

import (
	"log/slog"
	"time"
)

type WatchDog struct {
	pingChan chan struct{}
	stopChan chan struct{}
	done     chan struct{}
	Timeout  chan bool
}

/* Start new Watchdog timer with a given interval. Returns WatchDog struct */
func Start(duration float64) WatchDog {
	slog.Info("[Watchdog] Starting...")

	interval := time.Duration(duration * float64(time.Second))

	wd := WatchDog{
		pingChan: make(chan struct{}, 1),
		stopChan: make(chan struct{}),
		done:     make(chan struct{}),
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

// Stops Signals the watchdog timer to stop
func (wd *WatchDog) Stop() {
	select {
	case <-wd.done:
		return
	default:
		close(wd.stopChan)
		<-wd.done
	}
}

// Ping resets the watchdog timer
func (wd *WatchDog) Ping() {
	wd.pingChan <- struct{}{}
}
