package statesync

import (
	"fmt"
	"log"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
)

func ValidateStateWv(wv *Worldview) error {
	return nil
}

// ValidateStateRemote does sanity check on a remote elevator state
func ValidateStateRemote(res *RemoteElevatorState) error {
	isMoving := res.Behavior == elevator.BMoving
	isDoorOpen := res.DoorState == elevator.DSOpen

	if isMoving && isDoorOpen {
		return fmt.Errorf("cannot move with door open")
	}

	if res.TargetFloor < 0 || res.TargetFloor >= res.NumFloors {
		log.Printf("Target: %v, numFloors %v", res.TargetFloor, res.NumFloors)
		return fmt.Errorf("target floor %d is out of bounds", res.TargetFloor)
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
