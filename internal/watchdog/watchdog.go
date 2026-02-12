package watchdog

import (
	"log/slog"
	"time"
)

type WatchDog struct {
	pingChan chan struct{}
	stopChan chan struct{}
	done	 chan struct{}
	Timeout  chan bool
}


/* Start new Watchdog timer with a given interval. Returns WatchDog struct */
func Start(duration float64) WatchDog {
	slog.Info("[Watchdog] Starting...")

	interval := time.Duration(duration * float64(time.Second))
	
	wd := WatchDog{
		pingChan:	make(chan struct{}, 1),
		stopChan:	make(chan struct{}, 1),
		done:		make(chan struct{}, 1),
		Timeout:	make(chan bool, 1),

	}

	go func(){
		slog.Info("[Watchdog] Started")
		timer := time.NewTimer(interval)
		defer timer.Stop()
		defer close(wd.done)

		for {
			select {
			case <- timer.C:
				slog.Error("[Watchdog] Timed out")
				select {
				case wd.Timeout <- true:
				default:
				}
				return

			case <- wd.pingChan:
				slog.Info("[Watchdog] Ping received -> Timer is reset.")
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(interval)

			case <- wd.stopChan:
				slog.Info("[Watchdog] Timer Stopped")
				return
			}
		}
	}()

	return wd
}

/* Signals the watchdog timer to stop*/
func Stop(wd *WatchDog) {
	close(wd.stopChan)
	<-wd.done
}

/* Resets the watchdog timer */
func Ping(wd *WatchDog) {
	select {
	case wd.pingChan <- struct{}{}:
	default:
	}
}

