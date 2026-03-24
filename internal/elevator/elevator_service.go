// Package elevator provides interfacing with the motor and panel button lights.
package elevator

import (
	"fmt"
	"slices"

	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
)

type ElevatorService struct {
	io elevio.ElevatorDriver
}

// NewElevatorService returns the constructed ElevatorService struct.
func NewElevatorService(driver elevio.ElevatorDriver) ElevatorService {
	return ElevatorService{
		io: driver,
	}
}

// SetMoveDirection sets the motor direction to equal the intended direction of movement.
func (es *ElevatorService) SetMoveDirection(direction elevio.MotorDirection) error {
	legalDirections := []elevio.MotorDirection{elevio.MotorDirectionUp, elevio.MotorDirectionDown}

	if !slices.Contains(legalDirections, direction) {
		return fmt.Errorf("[ElevatorService] Got an invalid direction, Received: %v", direction)
	}

	es.io.SetMotorDirection(direction)
	return nil
}

// StopMotor disables the motor.
func (es *ElevatorService) StopMotor() {
	es.io.SetMotorDirection(elevio.MotorDirectionStop)
}

// SetDoorState sets the door state.
func (es *ElevatorService) SetDoorState(state bool) {
	es.io.SetDoorOpenLamp(state)
}

// ClearAllLights disables all panel button lights, excluding the stop light.
func (es *ElevatorService) ClearAllLights(amountOfFloors int) {
	for floor := range amountOfFloors {
		es.io.SetButtonLamp(elevio.Cab, floor, false)
		es.io.SetButtonLamp(elevio.HallUp, floor, false)
		es.io.SetButtonLamp(elevio.HallDown, floor, false)
	}
}

// SetStopLight toggles the stop light.
func (es *ElevatorService) SetStopLight(state bool) {
	es.io.SetStopLamp(state)
}

// SetCallLight toggles the light of the corresponding button.
func (es *ElevatorService) SetCallLight(buttonType elevio.ButtonType, floor int, state bool) {
	es.io.SetButtonLamp(buttonType, floor, state)
}

// SetCurrentFloorLight turns on the light for a given floor indicator.
func (es *ElevatorService) SetCurrentFloorLight(floor int) {
	es.io.SetFloorIndicator(floor)
}

// SetAllLights syncs all button lamps to the given call state.
// hallCalls[floor][0] = HallDown light (matches statesync.HDDown=0),
// hallCalls[floor][1] = HallUp light (matches statesync.HDUp=1).
// The caller is responsible for converting statesync types to [][2]bool before calling this.
func (es *ElevatorService) SetAllLights(numFloors int, cabCalls []bool, hallCalls [][2]bool) {
	es.SetCabCallLights(numFloors, cabCalls)
	es.SetHallCallLights(numFloors, hallCalls)
}

// SetCabCallLights turns on the lights for all active cab calls.
func (es *ElevatorService) SetCabCallLights(numFloors int, cabCalls []bool) {
	for floor := 0; floor < numFloors; floor++ {
		es.io.SetButtonLamp(elevio.Cab, floor, cabCalls[floor])
	}
}

// SetHallCallLights turns on the lights for all active hall calls.
func (es *ElevatorService) SetHallCallLights(numFloors int, hallCalls [][2]bool) {
	for floor := 0; floor < numFloors; floor++ {
		es.io.SetButtonLamp(elevio.HallDown, floor, hallCalls[floor][0])
		es.io.SetButtonLamp(elevio.HallUp, floor, hallCalls[floor][1])
	}
}
