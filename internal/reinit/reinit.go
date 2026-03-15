package reinit

import (
	"log/slog"
	"os"
	"syscall"
)

// Reinitialize restarts the program in the same terminal with identical arguments.
func Reinitialize() {
	exe, err := os.Executable()
	if err != nil {
		slog.Error("[Reinit] Failed to get executable path", "error", err)
	}
	args := os.Args

	slog.Info("[Reinit] Program reinitialization initiated. Restarting...")

	if err := syscall.Exec(exe, args, os.Environ()); err != nil {
		slog.Error("[Reinit] Failed to execute program", "error", err)
	}
}

// ErrorHandler calls on Reinitialization() on any errNo received.
func ErrorHandler(errCh <-chan int) {
	for {
		select {
		case <-errCh:
			slog.Warn("[ErrorHandler] Fatal Error detected -> Reinitializing")
			Reinitialize()
		}
	}
}
