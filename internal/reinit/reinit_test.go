package reinit

import (
	_ "log/slog"
	"os/exec"
	"testing"

	"github.com/Mosazghi/elevator-ttk4145/internal/watchdog"
)

// Reinitialization triggers full program reboot. Restarts with same parameters as original
func TestReinitialization(t *testing.T) {
	wd := watchdog.Start(3)
	defer wd.Stop()
	for {
		select {
		case <-wd.Timeout:
			t.Log("[Overwatch] Program reinitialization initiated. \n Restarting...")
			exec.Command("gnome-terminal", "--", "go", "run", "watchdog_test.go").Run()
			return
		default:
			continue
		}
	}

}
