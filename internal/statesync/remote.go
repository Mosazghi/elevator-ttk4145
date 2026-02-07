package statesync

import (
	"fmt"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
)

type RemoteElevatorState struct {
	ID           int                   `json:"id"`
	TargetFloor  int                   `json:"target_floor"`
	CurrentFloor int                   `json:"current_floor"`
	Direction    elevio.MotorDirection `json:"direction"`
	DoorState    elevator.DoorState    `json:"door_state"`
	CabCalls     []bool                `json:"cab_calls"`
	Behavior     elevator.Behavior     `json:"behavior"`
	LastSeenAt   time.Time             `json:"last_seen_at"`
	NumFloors    int                   `json:"num_floors"`
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

func (res RemoteElevatorState) String() string {
	return fmt.Sprintf("RemoteElevatorState{ID: %d, TargetFloor: %d, CurrentFloor: %d, Direction: %v, DoorState: %v, CabCalls: %v, Behavior: %v, LastSeenAt: %v}",
		res.ID, res.TargetFloor, res.CurrentFloor, res.Direction, res.DoorState, res.CabCalls, res.Behavior, res.LastSeenAt.Format(time.DateTime))
}
