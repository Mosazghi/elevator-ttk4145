package watchdog

import (
	"testing"
	"time"
)

// TestWatchdogTimeout verifies timeout when no ping is received.
func TestWatchdogTimeout(t *testing.T) {
	wd := New(100 * time.Millisecond)
	defer wd.Stop()
	go wd.Start()

	select {
	case <-wd.TimeoutChan:
		// Expected timeout
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Watchdog did not timeout")
	}
}

// TestWatchdogPing verifies periodic pings prevent timeout.
func TestWatchdogPing(t *testing.T) {
	wd := New(200 * time.Millisecond)

	defer wd.Stop()
	go wd.Start()

	done := make(chan bool)
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(100 * time.Millisecond)
			wd.Ping()
		}
		done <- true
	}()

	select {
	case <-wd.TimeoutChan:
		t.Fatal("Watchdog timed out despite pinging")
	case <-done:
	}
}

// TestWatchdogStop verifies Stop is safe to call multiple times.
func TestWatchdogStop(t *testing.T) {
	wd := New(1.0 * time.Second)
	defer wd.Stop()
	go wd.Start()

	wd.Stop()

	wd.Stop()
}

// TestWatchdogTimeoutAfterPings verifies timeout after pings stop.
func TestWatchdogTimeoutAfterPings(t *testing.T) {
	wd := New(150 * time.Millisecond)
	defer wd.Stop()
	go wd.Start()

	wd.Ping()
	time.Sleep(50 * time.Millisecond)
	wd.Ping()

	select {
	case <-wd.TimeoutChan:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Watchdog did not timeout after pings stopped")
	}
}
