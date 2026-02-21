package orders

import (
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// HasOrdersAbove checks if local elevator has active/processing orders above
func HasOrdersAbove(e *statesync.RemoteElevatorState, calls Calls) bool {
	for f := e.CurrentFloor + 1; f < len(calls.CabCalls); f++ {
		for b := range calls.HallCalls[f] {
			if calls.HallCalls[f][b].By == e.ID && calls.HallCalls[f][b].State == statesync.HSProcessing {
				return true
			}
		}
	}

	for f := e.CurrentFloor + 1; f < len(e.CabCalls); f++ {
		if e.CabCalls[f] {
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
				return true
			}
		}
	}
	for f := 0; f < e.CurrentFloor; f++ {
		if calls.CabCalls[f] {
			return true
		}
	}

	return false
}

// ShouldStop checks if local elevator should stop based current orders
func ShouldStop(e *statesync.RemoteElevatorState, calls Calls) bool {
	switch e.Direction {
	case elevio.MDDown:
		return calls.HallCalls[e.CurrentFloor][statesync.HDDown].State == statesync.HSProcessing ||
			calls.CabCalls[e.CurrentFloor] ||
			!HasOrdersBelow(e, calls)
	case elevio.MDUp:

		return calls.HallCalls[e.CurrentFloor][statesync.HDUp].State == statesync.HSProcessing ||
			calls.CabCalls[e.CurrentFloor] ||
			!HasOrdersAbove(e, calls)
	case elevio.MDStop:
		return true
	}

	return true
}
