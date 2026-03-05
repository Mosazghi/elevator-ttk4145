package controller

import (
	"log/slog"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// int requests_shouldClearImmediately(Elevator e, int btn_floor, Button btn_type){
//     return
//         e.floor == btn_floor &&
//         (
//             (e.dirn == D_Up   && btn_type == B_HallUp)    ||
//             (e.dirn == D_Down && btn_type == B_HallDown)  ||
//             e.dirn == D_Stop ||
//             btn_type == B_Cab
//         );
// }

func ShouldClearImmediately(e statesync.RemoteElevatorState, btnFloor int, btnType elevio.ButtonType) bool {
	if e.CurrentFloor != btnFloor {
		return false
	}

	switch e.Direction {
	case elevio.MDUp:
		return btnType == elevio.HallUp || btnType == elevio.Cab
	case elevio.MDDown:
		return btnType == elevio.HallDown || btnType == elevio.Cab
	case elevio.MDStop:
		fallthrough
	default:
		return true
	}
}

// HasOrdersAbove checks if local elevator has active/processing orders above
func HasOrdersAbove(e statesync.RemoteElevatorState, calls Calls) bool {
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
func HasOrdersBelow(e statesync.RemoteElevatorState, calls Calls) bool {
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
func ShouldStop(e statesync.RemoteElevatorState, calls Calls) bool {
	floor := e.CurrentFloor
	dir := e.Direction

	switch dir {
	case elevio.MDDown:
		hasDownCall := calls.HallCalls[floor][statesync.HDDown].State == statesync.HSProcessing
		hasCabCall := calls.CabCalls[floor]
		hasNoOrdersBelow := !HasOrdersBelow(e, calls)

		shouldStop := hasDownCall || hasCabCall || hasNoOrdersBelow

		return shouldStop

	case elevio.MDUp:
		hasUpCall := calls.HallCalls[floor][statesync.HDUp].State == statesync.HSProcessing
		hasCabCall := calls.CabCalls[floor]
		hasNoOrdersAbove := !HasOrdersAbove(e, calls)

		shouldStop := hasUpCall || hasCabCall || hasNoOrdersAbove

		return shouldStop

	case elevio.MDStop:
		fallthrough
	default:
		return true
	}
}

// ClearAtCurrentFloor clears hall and cab call requests at the elevator's current floor.
func ClearAtCurrentFloor(wv *statesync.Worldview, e statesync.RemoteElevatorState, calls *Calls) {
	floor := e.CurrentFloor

	wv.SetCabCall(floor, false)

	noHallCallsAt := func(dir statesync.HallCallDir) bool {
		return calls.HallCalls[floor][dir].State != statesync.HSProcessing && calls.HallCalls[floor][dir].By != e.ID
	}

	myHallCallsAt := func(dir statesync.HallCallDir) bool {
		return calls.HallCalls[floor][dir].State == statesync.HSProcessing && calls.HallCalls[floor][dir].By == e.ID
	}

	switch e.Direction {
	case elevio.MDUp:
		if myHallCallsAt(statesync.HDUp) {
			slog.Info("Completing my hall call at floor", "floor", floor, "dir", "up")
			wv.CompleteHallCall(floor, statesync.HDUp)
		}
		// Clear the down call if it's also assigned to us (simultaneous stop),
		// or if there are no orders above and no up call keeping us here.
		if myHallCallsAt(statesync.HDDown) || (!HasOrdersAbove(e, *calls) && noHallCallsAt(statesync.HDUp)) {
			slog.Info("Completing hall call at floor", "floor", floor, "dir", "down")
			wv.CompleteHallCall(floor, statesync.HDDown)
		}
	case elevio.MDDown:
		if myHallCallsAt(statesync.HDDown) {
			slog.Info("Completing my hall call at floor", "floor", floor, "dir", "down")
			wv.CompleteHallCall(floor, statesync.HDDown)
		}
		// Clear the up call if it's also assigned to us (simultaneous stop),
		// or if there are no orders below and no down call keeping us here.
		if myHallCallsAt(statesync.HDUp) || (!HasOrdersBelow(e, *calls) && noHallCallsAt(statesync.HDDown)) {
			slog.Info("Completing hall call at floor", "floor", floor, "dir", "up")
			wv.CompleteHallCall(floor, statesync.HDUp)
		}
	case elevio.MDStop:
		fallthrough
	default:
		if myHallCallsAt(statesync.HDUp) {
			slog.Info("Completing my hall call at floor", "floor", floor, "dir", "up")
			wv.CompleteHallCall(floor, statesync.HDUp)
		}
		if myHallCallsAt(statesync.HDDown) {
			slog.Info("Completing my hall call at floor", "floor", floor, "dir", "down")
			wv.CompleteHallCall(floor, statesync.HDDown)
		}
	}
}
