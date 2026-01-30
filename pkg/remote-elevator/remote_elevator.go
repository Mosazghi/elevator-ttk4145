package remoteelevator

import (
	"time"

	"Heisern/pkg/elevator"
	"Heisern/pkg/elevio"
)

type RemoteElevatorState struct {
	ID           int
	TargetFloor  int
	PrevFloor    int
	CurrentFloor int
	Direction    elevio.MotorDirection
	DoorState    elevator.DoorState
	CabCalls     []bool
	Behavior     elevator.Behavior
	LastSeenAt   time.Time
}
