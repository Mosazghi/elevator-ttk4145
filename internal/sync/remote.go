package statesync

import (
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
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

// NewRemoteElevatorState creates a new instance of  RemoteElevatorState
func NewRemoteElevatorState(id, floor, numFloors int) RemoteElevatorState {
	return RemoteElevatorState{
		ID:           id,
		TargetFloor:  floor,
		PrevFloor:    floor - 1, // TODO!: fix this
		CurrentFloor: floor,
		Direction:    elevio.Stop,
		DoorState:    elevator.DSClosed,
		CabCalls:     make([]bool, numFloors),
		Behavior:     elevator.BIdle,
		LastSeenAt:   time.Now(),
	}
}
