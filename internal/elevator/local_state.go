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
	BObstructed
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

func (e *ElevatorState) SetAction(action Action) error {
	if action.Behavior != BIdle &&
		action.Behavior != BMoving &&
		action.Behavior != BObstructed {
		return fmt.Errorf("[Elevator] Got an invalid behavior, Received: %v", action.Behavior)
	}

<<<<<<< HEAD
	if action.Direction != elevio.MDDown && action.Direction != elevio.MDUp && action.Direction != elevio.MDStop {
		return fmt.Errorf("[Elevator] Got an invalid direction, Received: %v", action.Direction)
=======
func (e *ElevState) SetDir(dir elevio.MotorDirection) {
	e.Dir = dir
	e.io.SetMotorDirection(dir)
}

func NewElevState(initFloor int, orders [4][3]bool, io elevio.ElevatorDriver) *ElevState {
	return &ElevState{
		io:        io,
		Target:    Order{-1, elevio.Cab},
		CurrFloor: initFloor,
		PrevFloor: -1,
		Dir:       elevio.MDStop,
		Behavior:  BIdle,
		Orders:    orders,
>>>>>>> 009012c (feat: TestGetNextOrder)
	}

	e.Behavior = action.Behavior
	e.Dir = action.Direction
	e.io.SetMotorDirection(action.Direction)
	return nil
}

func (e *ElevatorState) StopAction() {
	e.io.SetMotorDirection(elevio.MDStop)
}

<<<<<<< HEAD
// Continue the currently active order
func (e *ElevatorState) ContinueAction() {
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
=======
// ---- Event Handlers ----//

func (e *ElevState) OnInitBetweenFloors() {
	fmt.Println("Initializing: Between floors")
	e.SetDir(elevio.MDDown)
	e.Behavior = BMoving
}

func (e *ElevState) OnOrderRequest(order elevio.ButtonEvent) {
	fmt.Printf("[ORDER] %+v\n", order)
	e.io.SetButtonLamp(order.Button, order.Floor, true)
	e.Orders[order.Floor][order.Button] = true
	switch e.Behavior {
	case BIdle:
		// Mark as active

		// Set Target floor
		e.Target.RType = order.Button
		e.Target.Floor = order.Floor

		e.Dir, e.Behavior = ChooseDirection(e)

		e.io.SetMotorDirection(e.Dir)

	case BMoving:
	case BDoorOpen:
	}

	fmt.Printf("State: %v\n", e)
}

func (e *ElevState) OnNewFloorArrival(floor int) {
	fmt.Printf("[FLOOR] %+v\n", floor)
	fmt.Printf("STATE: %+v\n", e)
	// if floor == e.io.GetTotalFloors()-1 {
	// 	e.Dir = elevio.MD_Down
	// } else if floor == 0 {
	// 	e.Dir = elevio.MD_Up
	// }

	e.CurrFloor = floor
	e.io.SetFloorIndicator(e.CurrFloor)

	switch e.Behavior {
	case BMoving:
		if ShouldStop(e) {
			// stop
			e.Dir = elevio.MDStop
			e.io.SetMotorDirection(e.Dir)
			ClearAtCurrentFloor(e)
			e.SetAllLights()
			e.io.SetDoorOpenLamp(true)
			time.Sleep(3 * time.Second)
			e.io.SetDoorOpenLamp(false)
			e.Dir, e.Behavior = ChooseDirection(e)
		}
	}
}

func (e *ElevState) OnObstructionSignal(obstructed bool) {
	fmt.Printf("[OBSTR] %+v\n", obstructed)
	if obstructed {
		e.io.SetMotorDirection(elevio.MDStop)
>>>>>>> 009012c (feat: TestGetNextOrder)
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
