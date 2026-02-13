package reinit
// package main

import (
	"log/slog"
	"os/exec"

	"github.com/Mosazghi/elevator-ttk4145/internal/watchdog"
)

type WatchDog = watchdog.WatchDog

func Reinitialization() {
	wd := watchdog.Start(3)
	defer wd.Stop()
	for {
		select {
		case <-wd.Timeout:
			slog.Info("[Overwatch] Program reinitialization initiated. Restarting...")

			cmd := exec.Command("gnome-terminal", "--", "bash", "-c", "go run watchdog_test.go; exec bash")
			if err := cmd.Run(); err != nil {
				slog.Error("Failed to start new terminal", "error", err)
			}

		default:
			continue
		}
	}
}

func main() {
	Reinitialization()
}
