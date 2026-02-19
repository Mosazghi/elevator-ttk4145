package reinit

import (
	"log/slog"
	"os"
	"testing"
)

// Reinitialization restarts the program in the same terminal with identical arguments.
// TestReinitializationSetup tests that the executable and arguments are retrieved correctly.
func TestReinitializationSetup(t *testing.T) {
    exe, err := os.Executable()
    if err != nil {
        t.Error("[Reinit] Failed to get executable path", "error", err)
    }
    if exe == "" {
        t.Error("[Reinit] Executable path is empty")
    }
    args := os.Args
    if len(args) == 0 {
        t.Error("[Reinit] Arguments are empty")
    }
}

// ErrorHandler calls on Reinitialization() on any errNo received.
func TestErrorHandler(errCh <-chan int) {

	for {
		select {
		case <-errCh:
			slog.Warn("[ErrorHandler] Fatal Error detected -> Reinitializing")
			Reinitialization()
		}
	}

}


