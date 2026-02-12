// Package elevator
package elevator

import (
	"log/slog"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
)

type (
	Behavior   int
	DoorState  int
	LightState int
)

const (
	BIdle Behavior = iota
	BMoving
	BDoorOpen
	BObstructed
	BSize
)

const (
	DSClosed DoorState = iota
	DSOpen
)

const (
	LSOff LightState = iota
	LSOn
)

func (d DoorState) String() string {
	switch d {
	case DSClosed:
		return "CLOSED"
	case DSOpen:
		return "OPEN"
	}
	return "UNKNOWN"
}

func (l LightState) String() string {
	switch l {
	case LSOff:
		return "Off"
	case LSOn:
		return "On"
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
	SetCallLight(buttonType elevio.ButtonType, floor int, state bool)
	SetCurrentFloorLight(floor int)
	SetStopLight(state bool)
	SetDoor(state bool)
	String()
	Stop()
}

func (e *ElevatorState) SetAction(action Action) {
	if action.Behavior < 0 || action.Behavior >= BSize {
		slog.Error("[Elevator] Got an invalid behavior", "Received behavior", action.Behavior)
		return
	}

	if action.Direction != elevio.MDDown && action.Direction != elevio.MDUp && action.Direction != elevio.MDStop {
		slog.Error("[Elevator] Got an invalid direction", "Direction", action.Direction)
		return
	}

	e.Behavior = action.Behavior
	e.Dir = action.Direction
	e.io.SetMotorDirection(action.Direction)
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

// Return an int along getNextAction func to indicate light on/off
// Off happends when MDStop and BIdle while other is always on.

func (e *ElevatorState) SetDoor(state DoorState) {
	if e.Behavior != BIdle {
		slog.Error("[Elevator] Cannot open door when not idle", "current behavior", e.Behavior)
		return
	}

	if state == DSOpen {
		e.io.SetDoorOpenLamp(true)
	} else {
		e.io.SetDoorOpenLamp(false)
	}
}

func (e *ElevatorState) SetStopLight(state LightState) {
	if state == LSOff {
		e.io.SetStopLamp(false)
	} else {
		e.io.SetStopLamp(true)
	}
}

func (e *ElevatorState) SetCallLight(buttonType elevio.ButtonType, floor int, state LightState) {
	if state == LSOff {
		e.io.SetButtonLamp(buttonType, floor, false)
	} else {
		e.io.SetButtonLamp(buttonType, floor, true)
	}
}

func (e *ElevatorState) SetCurrentFloorLight(floor int) {
	e.io.SetFloorIndicator(floor)
}

func (e *ElevatorState) String() {
	slog.Info("[Elevator] Current ElevatorState: ", "behavior", e.Behavior, "Direction", e.Dir)
}

func NewElevator(behavior Behavior, direction elevio.MotorDirection, driver elevio.ElevatorDriver) ElevatorState {
	return ElevatorState{
		driver,
		direction,
		behavior,
	}
}
