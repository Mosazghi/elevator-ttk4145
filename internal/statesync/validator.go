package statesync

import (
	"fmt"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
)

// IsValidFloor validates a given floor.
func IsValidFloor(floor, maxFloors int) bool {
	if floor >= maxFloors || floor < 0 {
		return false
	}
	return true
}

// IsValidDirTransition checks if a transition from currentDir to newDir is valid.
func IsValidDirTransition(currentState, newState HallCallState) error {
	switch newState {
	case HallCallStateNone:
		if currentState != HallCallStateProcessing {
			return fmt.Errorf("cannot go to None from %+v", currentState)
		}

	case HallCallStateUnconfirmed:
		if currentState != HallCallStateNone {
			return fmt.Errorf("cannot go to Unconfirmed from %+v", currentState)
		}
	case HallCallStateConfirmed:
		if currentState != HallCallStateUnconfirmed && currentState != HallCallStateNone {
			return fmt.Errorf("cannot go to Confirmed from %+v", currentState)
		}

	case HallCallStateProcessing:
		if currentState != HallCallStateConfirmed {
			return fmt.Errorf("cannot go to Processing from %+v", currentState)
		}

	default:
		return fmt.Errorf("invalid hall call state: %+v", newState)
	}

	return nil
}

// ValidateStateRemote performs sanity checks on the remote elevator states.
func ValidateStateRemote(res *RemoteElevatorState) error {
	if res == nil {
		return fmt.Errorf("remote elevator state cannot be nil")
	}

	isMoving := res.Behavior == elevator.BehaviorMoving
	isDoorOpen := res.Behavior == elevator.BehaviorDoorOpen

	if isMoving && isDoorOpen {
		return fmt.Errorf("cannot move with door open")
	}

	// -1 because we could be inbetween floors
	if res.CurrentFloor < -1 || res.CurrentFloor >= res.NumFloors {
		return fmt.Errorf("current floor %d is out of bounds", res.CurrentFloor)
	}

	if len(res.CabCalls) != res.NumFloors {
		return fmt.Errorf("cab calls length %d does not match number of floors %d", len(res.CabCalls), res.NumFloors)
	}

	return nil
}
