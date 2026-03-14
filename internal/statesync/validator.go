// Package validate is used for validation of states related to worldview, floors, elevators etc.
package statesync

import (
	"fmt"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
)

// IsValidFloor validates a given floor
func IsValidFloor(floor, maxFloors int) bool {
	if floor >= maxFloors || floor < 0 {
		return false
	}
	return true
}

// IsValidDirTransition checks if a transition from currDir to newDir is valid
func IsValidDirTransition(currState, newState HallCallState) error {
	switch currState {
	case HallCallStateConfirmed:
		if newState == HallCallStateNone {
			return fmt.Errorf("cannot go to Available from None")
		}

	case HallCallStateUnconfirmed:
		if newState == HallCallStateConfirmed {
			return fmt.Errorf("cannot go to Processing from Available")
		}
	case HallCallStateNone:
		if newState == HallCallStateUnconfirmed {
			return fmt.Errorf("cannot go to None from Processing")
		}
	default:
		return fmt.Errorf("invalid hall call state: %v", newState)
	}

	return nil
}

// ValidateStateRemote does sanity check on a remote elevator state
func ValidateStateRemote(res *RemoteElevatorState) error {
	if res == nil {
		return fmt.Errorf("remote elevator state cannot be nil")
	}

	isMoving := res.Behavior == elevator.BMoving
	isDoorOpen := res.DoorState == elevator.DSOpen

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
