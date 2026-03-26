package statesync

import (
	"fmt"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	"github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	"github.com/Mosazghi/elevator-ttk4145/pkg/shared"
)

// RemoteElevatorState represents the state of a remote elevator (node) in the system.
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
	TimedOutAt   time.Time             `json:"timed_out_at"`
}

// NewRemoteElevatorState constructs a new instance of RemoteElevatorState.
func NewRemoteElevatorState(id, numFloors int) *RemoteElevatorState {
	return &RemoteElevatorState{
		ID:           id,
		CurrentFloor: -1,
		Direction:    elevio.MotorDirectionStop,
		CabCalls:     make([]bool, numFloors),
		Behavior:     elevator.BehaviorIdle,
		LastSeenAt:   time.Now(),
		NumFloors:    numFloors,
		Alive:        true,
		TimedOutAt:   time.Time{},
	}
}

// String returns a string representation of the RemoteElevatorState.
func (res *RemoteElevatorState) String() string {
	return fmt.Sprintf("RemoteElevatorState{ID: %d, CurrentFloor: %d, Direction: %v, DoorState: %v, CabCalls: %v, Behavior: %v, LastSeenAt: %v}",
		res.ID, res.CurrentFloor, res.Direction, res.DoorState, res.CabCalls, res.Behavior, res.LastSeenAt.Format(time.DateTime))
}

// IsAllowedToServe checks if an elevator is eligible to accept new hall calls.
func (res *RemoteElevatorState) IsAllowedToServe() bool {
	return (res.Alive && !res.IsObstructed) &&
		time.Since(res.TimedOutAt) > blockNewOrderDuration
}

// IsOppositeHallCallDirection returns the direction opposite of travel.
func (res *RemoteElevatorState) IsOppositeHallCallDirection(hallCallDirection HallCallDirection) bool {
	var direction elevio.MotorDirection
	if hallCallDirection == HallCallDirectionUp {
		direction = elevio.MotorDirectionUp
	} else {
		direction = elevio.MotorDirectionDown
	}

	return res.Direction != direction
}

// IsOppositeMotorDirection returns the direction opposite of the motor.
func (res *RemoteElevatorState) IsOppositeMotorDirection(direction elevio.MotorDirection) bool {
	return res.Direction != direction
}

// UndefinedState checks if the elevator is out of bounds.
func (res *RemoteElevatorState) UndefinedState() bool {
	return res.CurrentFloor == shared.UndefinedFloor && res.Direction != elevio.MotorDirectionStop
}
