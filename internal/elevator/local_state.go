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

type Action struct {
	Behavior  Behavior
	Direction elevio.MotorDirection
}

type ElevatorCallbacks interface {
	SetAction(behavior Behavior, direction elevio.MotorDirection)
	Stop()
	OnInitBetweenFloors()
}

func (e *ElevatorState) SetAction(action Action) error {
	if action.Behavior < 0 || action.Behavior >= BSize {
		return errors.New("got an invalid behavior")
	}

	if action.Direction != elevio.MDDown && action.Direction != elevio.MDUp && action.Direction != elevio.MDStop {
		return errors.New("got an invalid direction")
	}

	e.Behavior = action.Behavior
	e.Dir = action.Direction
	e.io.SetMotorDirection(action.Direction)
	return nil
}

func (e *ElevatorState) Stop() {
	e.io.SetMotorDirection(elevio.MDStop)
}

// Continue the currently active order
func (e *ElevatorState) Continue() {
	if e.Behavior == BMoving {
		e.io.SetMotorDirection(e.Dir)
	}
}

func NewElevator(behavior Behavior, direction elevio.MotorDirection, driver elevio.ElevatorDriver) ElevatorState {
	return ElevatorState{
		driver,
		direction,
		behavior,
	}
}
