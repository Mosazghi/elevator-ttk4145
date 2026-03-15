//go:build ignore

package watchdog

import (
	"testing"
	"time"
)

// TestWatchdogTimeout tests that the watchdog times out when not pinged
func TestWatchdogTimeout(t *testing.T) {
	wd := Start(0.1) // 100ms timeout
	defer wd.Stop()

	select {
	case <-wd.Timeout:
		// Expected timeout
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Watchdog did not timeout")
	}
}

// TestWatchdogPing tests that pinging prevents timeout
func TestWatchdogPing(t *testing.T) {
	wd := Start(0.2) // 200ms timeout
	defer wd.Stop()

	// Ping every 100ms for 500ms
	done := make(chan bool)
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(100 * time.Millisecond)
			wd.Ping()
		}
		done <- true
	}()

	select {
	case <-wd.Timeout:
		t.Fatal("Watchdog timed out despite pinging")
	case <-done:
		// Expected: no timeout
	}
}

// TestWatchdogStop tests that the watchdog can be stopped
func TestWatchdogStop(t *testing.T) {
	wd := Start(1.0)
	wd.Stop()

	// Stopping again should not panic
	wd.Stop()
}

// TestWatchdogTimeoutAfterPings tests timeout occurs after pings stop
func TestWatchdogTimeoutAfterPings(t *testing.T) {
	wd := Start(0.15) // 150ms timeout
	defer wd.Stop()

	wd.Ping()
	time.Sleep(50 * time.Millisecond)
	wd.Ping()

	// Now wait for timeout
	select {
	case <-wd.Timeout:
		// Expected timeout
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Watchdog did not timeout after pings stopped")
	}
}
