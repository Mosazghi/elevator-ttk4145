// Package elevator
package elevator

import (
	"fmt"
	"slices"

	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"
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
	DoorClosed DoorState = iota
	DoorOpen
)

const (
	LightOff LightState = iota
	LightOn
)

func (b Behavior) String() string {
	switch b {
	case BIdle:
		return "idle"
	case BMoving:
		return "moving"
	case BDoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}

func (d DoorState) String() string {
	switch d {
	case DoorClosed:
		return "closed"
	case DoorOpen:
		return "open"
	}
	return "unknown"
}

func (l LightState) String() string {
	switch l {
	case LightOff:
		return "off"
	case LightOn:
		return "on"
	}
	return "unknown"
}

type ElevatorService struct {
	io elevio.ElevatorDriver
}

func NewElevatorService(driver elevio.ElevatorDriver) ElevatorService {
	return ElevatorService{
		io: driver,
	}
}

func (e *ElevatorService) MoveDirection(direction elevio.MotorDirection) error {
	legalDirections := []elevio.MotorDirection{elevio.MDUp, elevio.MDDown}

	if !slices.Contains(legalDirections, direction) {
		return fmt.Errorf("[ElevatorService] Got an invalid direction, Received: %v", direction)
	}

	e.io.SetMotorDirection(direction)
	return nil
}

func (e *ElevatorService) Stop() {
	e.io.SetMotorDirection(elevio.MDStop)
}

func (e *ElevatorService) SetDoor(state bool) {
	if state {
		// TODO: PLAY SOUND?
	}
	e.io.SetDoorOpenLamp(state)
}

func (e *ElevatorService) ClearAllLights(amountOfFloors int) {
	for floor := range amountOfFloors {
		e.io.SetButtonLamp(elevio.Cab, floor, false)
		e.io.SetButtonLamp(elevio.HallUp, floor, false)
		e.io.SetButtonLamp(elevio.HallDown, floor, false)
	}
}

func (e *ElevatorService) SetStopLight(state LightState) {
	if state == LightOff {
		e.io.SetStopLamp(false)
	} else {
		e.io.SetStopLamp(true)
	}
}

func (e *ElevatorService) SetCallLight(buttonType elevio.ButtonType, floor int, state bool) {
	e.io.SetButtonLamp(buttonType, floor, state)
}

func (e *ElevatorService) SetCurrentFloorLight(floor int) {
	e.io.SetFloorIndicator(floor)
}

// SetAllLights syncs all button lamps to the given call state.
// hallCalls[floor][0] = HallDown light (matches statesync.HDDown=0),
// hallCalls[floor][1] = HallUp light (matches statesync.HDUp=1).
// The caller is responsible for converting statesync types to [][2]bool before calling this.
func (e *ElevatorService) SetAllLights(numFloors int, cabCalls []bool, hallCalls [][2]bool) {
	e.SetCabCallLights(numFloors, cabCalls)
	e.SetHallCallLights(numFloors, hallCalls)
}

// SetCabCallLights sets lights for all active cab calls
func (e *ElevatorService) SetCabCallLights(numFloors int, cabCalls []bool) {
	for floor := 0; floor < numFloors; floor++ {
		e.io.SetButtonLamp(elevio.Cab, floor, cabCalls[floor])
	}
}

// SetHallCallLights sets lights for all active hall calls
func (e *ElevatorService) SetHallCallLights(numFloors int, hallCalls [][2]bool) {
	for floor := 0; floor < numFloors; floor++ {
		e.io.SetButtonLamp(elevio.HallDown, floor, hallCalls[floor][0])
		e.io.SetButtonLamp(elevio.HallUp, floor, hallCalls[floor][1])
	}
}
