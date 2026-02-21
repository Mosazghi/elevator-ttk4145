package orders

import (
	"log/slog"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// HasOrdersAbove checks if local elevator has active/processing orders above
func HasOrdersAbove(e *statesync.RemoteElevatorState, calls Calls) bool {
	for f := e.CurrentFloor + 1; f < len(calls.CabCalls); f++ {
		for b := range calls.HallCalls[f] {
			if calls.HallCalls[f][b].By == e.ID && calls.HallCalls[f][b].State == statesync.HSProcessing {
				slog.Warn("Found hall call above",
					"floor", f,
					"direction", b,
					"by", calls.HallCalls[f][b].By,
					"state", calls.HallCalls[f][b].State)
				return true
			}
		}
	}

	for f := e.CurrentFloor + 1; f < len(e.CabCalls); f++ {
		if e.CabCalls[f] {
			slog.Warn("Found cab call above",
				"floor", f)
			return true
		}
	}
	return false
}

// HasOrdersAbove checks if local elevator has active/processing orders below
func HasOrdersBelow(e *statesync.RemoteElevatorState, calls Calls) bool {
	for f := 0; f < e.CurrentFloor; f++ {
		for b := range calls.HallCalls[f] {
			if calls.HallCalls[f][b].By == e.ID && calls.HallCalls[f][b].State == statesync.HSProcessing {
				// log
				slog.Warn("Found hall call below",
					"floor", f,
					"direction", b,
					"by", calls.HallCalls[f][b].By,
					"state", calls.HallCalls[f][b].State)
				return true
			}
		}
	}
	for f := 0; f < e.CurrentFloor; f++ {
		if calls.CabCalls[f] {
			slog.Warn("Found cab call below",
				"floor", f)
			return true
		}
	}

	return false
}

// ShouldStop checks if local elevator should stop based current orders
func ShouldStop(e *statesync.RemoteElevatorState, calls Calls) bool {
	floor := e.CurrentFloor
	dir := e.Direction

	slog.Warn("Checking if should stop",
		"floor", floor,
		"direction", dir)

	switch dir {
	case elevio.MDDown:
		hasDownCall := calls.HallCalls[floor][statesync.HDDown].State == statesync.HSProcessing
		hasCabCall := calls.CabCalls[floor]
		hasNoOrdersBelow := !HasOrdersBelow(e, calls)

		shouldStop := hasDownCall || hasCabCall || hasNoOrdersBelow

		slog.Warn("Down direction stop check",
			"hasDownCall", hasDownCall,
			"hasCabCall", hasCabCall,
			"hasNoOrdersBelow", hasNoOrdersBelow,
			"shouldStop", shouldStop)

		return shouldStop

	case elevio.MDUp:
		hasUpCall := calls.HallCalls[floor][statesync.HDUp].State == statesync.HSProcessing
		hasCabCall := calls.CabCalls[floor]
		hasNoOrdersAbove := !HasOrdersAbove(e, calls)

		shouldStop := hasUpCall || hasCabCall || hasNoOrdersAbove

		slog.Warn("Up direction stop check",
			"hasUpCall", hasUpCall,
			"hasCabCall", hasCabCall,
			"hasNoOrdersAbove", hasNoOrdersAbove,
			"shouldStop", shouldStop)

		return shouldStop

	case elevio.MDStop:
		slog.Warn("Already stopped, should stop")
		return true
	default:
		slog.Warn("Unknown direction, defaulting to stop")
		return true
	}
}
