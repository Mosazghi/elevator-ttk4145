// Package elevator
package elevator

import (
	"errors"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
)

type (
	Behavior  int
	DoorState int
)

const (
	BIdle Behavior = iota
	BMoving
	BDoorOpen
	BObstructed
	BSize
)

const (
	DSClosing DoorState = iota
	DSClosed
	DSOpening
	DSOpen
)

func (d DoorState) String() string {
	switch d {
	case DSClosing:
		return "CLOSING"
	case DSClosed:
		return "CLOSED"
	case DSOpening:
		return "OPENING"
	case DSOpen:
		return "OPEN"
	}
	return "UNKNOWN"
}

func (b Behavior) String() string {
	switch b {
	case BIdle:
		return "IDLE"
	case BMoving:
		return "MOVING"
	case BDoorOpen:
		return "DOOR_OPEN"
	}
	return "UNKNOWN"
}

type ElevatorState struct {
	io       elevio.ElevatorDriver
	Dir      elevio.MotorDirection
	Behavior Behavior
}

type ElevatorCallbacks interface {
	OnStopSignal(signal bool)
	SetAction(behavior Behavior, direction elevio.MotorDirection)

	OnInitBetweenFloors()
}

func (e *ElevatorState) SetAction(behavior Behavior, direction elevio.MotorDirection) error {
	if behavior < 0 || behavior >= BSize {
		return errors.New("got an invalid behavior")
	}

	if direction != elevio.MDDown && direction != elevio.MDUp && direction != elevio.MDStop {
		return errors.New("got an invalid direction")
	}

	e.Behavior = behavior
	e.Dir = direction
	e.io.SetMotorDirection(direction)
	return nil
}

func (e *ElevatorState) OnStopSignal(signal bool) {
}

func NewElevator(behavior Behavior, direction elevio.MotorDirection, driver *elevio.ElevIoDriver) ElevatorState {
	return ElevatorState{
		driver,
		direction,
		behavior,
	}
}
