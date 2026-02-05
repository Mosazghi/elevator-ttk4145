// Package validate is used for validation of states related to worldview, floors, elevators etc.
package statesync

import (
	"fmt"
)

// IsValidFloor validates a given floor
func IsValidFloor(floor, maxFloors int) bool {
	if floor > maxFloors || floor < 0 {
		return false
	}
	return true
}

func IsValidDirTransition(currDir, newDir HallCallState) error {
	switch currDir {
	case HSAvailable:
		if newDir == HSNone {
			return fmt.Errorf("invalid state transition: cannot go to Available from None")
		}

	case HSProcessing:
		if newDir == HSAvailable {
			return fmt.Errorf("invalid state transition: cannot go to Processing from Available")
		}
	case HSNone:
		if newDir == HSProcessing {
			return fmt.Errorf("invalid state transition: cannot go to None from Processing")
		}
	default:
		return fmt.Errorf("invalid hall call state: %v", newDir)
	}

	return nil
}
