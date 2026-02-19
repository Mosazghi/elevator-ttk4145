package reinit

import (
	_ "log/slog"
	"os"
	"syscall"
	"testing"

	"github.com/Mosazghi/elevator-ttk4145/internal/watchdog"
)

// Reinitialization triggers full program reboot. Restarts with same parameters as original
func TestReinitialization(t *testing.T) {
	
	exe, err := os.Executable()
	if err != nil {
		t.Error("[Overwatch] Failed to get executable path", "error", err)
	}
	args := os.Args

	wd := watchdog.Start(3)
	defer wd.Stop()
	for {
		select {
		case <-wd.Timeout:
			t.Log("[Overwatch] Program reinitialization initiated. \n Restarting...")
			if err := syscall.Exec(exe, args, os.Environ()); err != nil {
				t.Error("[Overwatch] Failed to execute program", "error", err)
			}
			//exec.Command("gnome-terminal", "--", "go", "run", "watchdog_test.go").Run()
			return
		default:
			continue
		}
	}

}
