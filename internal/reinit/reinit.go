package reinit

import (
	"flag"
	"log/slog"
	"os"
	"syscall"

	"github.com/Mosazghi/elevator-ttk4145/internal/watchdog"
)

type WatchDog = watchdog.WatchDog

func Reinitialization() {

	id := flag.Int("id", 0, "Node ID")
	port := flag.Int("port", 5000, "Broadcast port")

	flag.Parse()
	slog.Info("initialized", "id", *id, "port", *port)

	exe, err := os.Executable()
	if err != nil {
		slog.Error("[Overwatch] Failed to get executable path", "error", err)
	}
	args := os.Args

	wd := watchdog.Start(3)
	defer wd.Stop()
	for {
		select {
		case <-wd.Timeout:
			slog.Info("[Overwatch] Program reinitialization initiated. Restarting...")

			if err := syscall.Exec(exe, args, os.Environ()); err != nil {
				slog.Error("[Overwatch] Failed to execute program", "error", err)
			}

		default:
			continue
		}
	}
}
