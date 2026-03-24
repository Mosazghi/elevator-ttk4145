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
		slog.Error("failed to get executable path", "error", err)
	}
	args := os.Args

	slog.Info("program reinitialization initiated. Restarting...")

	if err := syscall.Exec(exe, args, os.Environ()); err != nil {
		slog.Error("failed to execute program", "error", err)
	}
}
