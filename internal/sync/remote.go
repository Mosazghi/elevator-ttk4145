package statesync

import (
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
)

type RemoteElevatorState struct {
	ID           int
	TargetFloor  int
	CurrentFloor int
	Direction    elevio.MotorDirection
	DoorState    elevator.DoorState
	CabCalls     []bool
	Behavior     elevator.Behavior
	LastSeenAt   time.Time
	NumFloors    int
}

// NewRemoteElevatorState creates a new instance of  RemoteElevatorState
func NewRemoteElevatorState(id, numFloors int) *RemoteElevatorState {
	return &RemoteElevatorState{
		ID:           id,
		TargetFloor:  0,
		CurrentFloor: -1,
		Direction:    elevio.Stop,
		DoorState:    elevator.DSClosed,
		CabCalls:     make([]bool, numFloors),
		Behavior:     elevator.BIdle,
		LastSeenAt:   time.Now(),
		NumFloors:    numFloors,
	}
}
