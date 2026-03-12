// Package elevator
package elevator

import (
	"fmt"
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

type ElevatorCallbacks interface {
	DoMotorAction(action MoveAction)
	SetCallLight(buttonType elevio.ButtonType, floor int, state bool)
	SetCurrentFloorLight(floor int)
	SetStopLight(state bool)
	SetDoor(state DoorState)
	String()
	Stop()
}

func (e *ElevatorState) DoMotorAction(action MoveAction) error {
	if action.Direction != elevio.MDDown && action.Direction != elevio.MDUp && action.Direction != elevio.MDStop {
		return fmt.Errorf("[Elevator] Got an invalid direction, Received: %v", action.Direction)
	}

	e.Behavior = action.Behavior
	e.Dir = action.Direction
	e.io.SetMotorDirection(e.Dir)
	return nil
}

func (e *ElevatorState) Stop() {
	e.io.SetMotorDirection(elevio.MDStop)
}

// ContinueLastDir is used to continue in the last direction after a stop
func (e *ElevatorState) ContinueLastDir() {
	if e.Behavior == BMoving {
		e.io.SetMotorDirection(e.Dir)
	}
}

// OnInitBetweenFloors is called when the elevator is initialized between floors
func (e *ElevatorState) OnInitBetweenFloors() {
	e.io.SetMotorDirection(elevio.MDDown)
	e.Behavior = BMoving
	e.Dir = elevio.MDDown
}

// Return an int along getNextAction func to indicate light on/off
// Off happends when MDStop and BIdle while other is always on.

func (e *ElevatorState) SetDoor(state bool) {
	if state {
		// TODO: PLAY SOUND?
	}
	e.io.SetDoorOpenLamp(state)
}

func (e *ElevatorState) SetStopLight(state LightState) {
	if state == LSOff {
		e.io.SetStopLamp(false)
	} else {
		e.io.SetStopLamp(true)
	}
}

func (e *ElevatorState) SetCallLight(buttonType elevio.ButtonType, floor int, state bool) {
	e.io.SetButtonLamp(buttonType, floor, state)
}

func (e *ElevatorState) SetCurrentFloorLight(floor int) {
	e.io.SetFloorIndicator(floor)
}

// SetAllLights syncs all button lamps to the given call state.
// hallCalls[floor][0] = HallDown light (matches statesync.HDDown=0),
// hallCalls[floor][1] = HallUp light (matches statesync.HDUp=1).
// The caller is responsible for converting statesync types to [][2]bool before calling this.
func (e *ElevatorState) SetAllLights(numFloors int, cabCalls []bool, hallCalls [][2]bool) {
	for floor := 0; floor < numFloors; floor++ {
		e.io.SetButtonLamp(elevio.HallDown, floor, hallCalls[floor][0])
		e.io.SetButtonLamp(elevio.HallUp, floor, hallCalls[floor][1])
		e.io.SetButtonLamp(elevio.Cab, floor, cabCalls[floor])
	}
}

func (e *ElevatorState) String() {
	slog.Info("[Elevator] Current ElevatorState: ", "behavior", e.Behavior, "Direction", e.Dir)
}

func NewElevator(behavior Behavior, direction elevio.MotorDirection, driver elevio.ElevatorDriver) ElevatorState {
	return ElevatorState{
		io:       driver,
		Dir:      direction,
		Behavior: behavior,
	}
}
