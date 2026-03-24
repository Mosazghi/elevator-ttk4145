package statesync

import (
	"fmt"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"
)

const UndefinedFloor = -1

type RemoteElevatorState struct {
	ID           int                   `json:"id"`
	CurrentFloor int                   `json:"current_floor"`
	IsObstructed bool                  `json:"is_obstructed"`
	Direction    elevio.MotorDirection `json:"direction"`
	DoorState    elevator.DoorState    `json:"door_state"`
	CabCalls     []bool                `json:"cab_calls"`
	Behavior     elevator.Behavior     `json:"behavior"`
	LastSeenAt   time.Time             `json:"last_seen_at"`
	NumFloors    int                   `json:"num_floors"`
	Alive        bool                  `json:"alive"`
	TimedOutAt	 time.Time			   `json:"timed_out_at"`
}

// NewRemoteElevatorState creates a new instance of  RemoteElevatorState
func NewRemoteElevatorState(id, numFloors int) *RemoteElevatorState {
	return &RemoteElevatorState{
		ID:           id,
		CurrentFloor: -1,
		Direction:    elevio.MDStop,
		DoorState:    elevator.DoorClosed,
		CabCalls:     make([]bool, numFloors),
		Behavior:     elevator.BIdle,
		LastSeenAt:   time.Now(),
		NumFloors:    numFloors,
		Alive:        true,
		TimedOutAt:   time.Time{},
	}
}

func (res *RemoteElevatorState) String() string {
	return fmt.Sprintf("RemoteElevatorState{ID: %d, CurrentFloor: %d, Direction: %v, DoorState: %v, CabCalls: %v, Behavior: %v, LastSeenAt: %v}",
		res.ID, res.CurrentFloor, res.Direction, res.DoorState, res.CabCalls, res.Behavior, res.LastSeenAt.Format(time.DateTime))
}

func (res *RemoteElevatorState) AllowedToServe() bool {
	return (res.Alive && !res.IsObstructed) &&
	time.Since(res.TimedOutAt) > BlockNewOrderDuration 
}

func (res *RemoteElevatorState) IsOppositeHallCallDirection(hallCallDirection HallCallDir) bool {
	var direction elevio.MotorDirection
	if hallCallDirection == HDUp {
		direction = elevio.MDUp
	} else {
		direction = elevio.MDDown
	}

	return res.Direction != direction
}

func (res *RemoteElevatorState) IsOppositeMotorDirection(direction elevio.MotorDirection) bool {
	return res.Direction != direction
}

func (res *RemoteElevatorState) UndefinedState() bool {
	return res.CurrentFloor == UndefinedFloor && res.Direction != elevio.MDStop
}
