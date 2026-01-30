package remoteelevator

import (
	"time"

	"SingleElevator/pkg/elevator"
	"SingleElevator/pkg/elevio"
)

type RemoteElevatorState struct {
	Id           int
	TargetFloor  int
	PrevFloor    int
	CurrentFloor int
	Direction    elevio.MotorDirection
	DoorState    elevator.DoorState
	CabCalls     []bool
	Behavior     elevator.Behavior
	LastSeenAt   time.Time
}
