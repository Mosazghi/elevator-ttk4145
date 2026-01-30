// Package elevator
package elevator

import (
	"fmt"
	"time"

	"Heisern/pkg/elevio"
)

type (
	Behavior  int
	DoorState int
)

const (
	Idle Behavior = iota
	Moving
	DoorOpen
)

const (
	Stuck DoorState = iota
	Open
	Closed
)

func (b Behavior) String() string {
	switch b {
	case Idle:
		return "IDLE"
	case Moving:
		return "MOVING"
	case DoorOpen:
		return "DOOR_OPEN"
	}
	return "UNKNOWN"
}

type Order struct {
	Floor int
	RType elevio.ButtonType
}

type ElevState struct {
	io        elevio.ElevatorDriver
	Target    Order
	CurrFloor int
	PrevFloor int
	Dir       elevio.MotorDirection
	Behavior  Behavior
	Orders    [4][3]bool
}

func (e *ElevState) ClearAllOrders() {
	for f := range e.Orders {
		for b := range e.Orders[f] {
			e.Orders[f][b] = false
			e.io.SetButtonLamp(elevio.ButtonType(b), f, false)
		}
	}
}

func (e *ElevState) SetDir(dir elevio.MotorDirection) {
	e.Dir = dir
	e.io.SetMotorDirection(dir)
}

func NewElevState(initFloor int, orders [4][3]bool, io elevio.ElevatorDriver) ElevState {
	return ElevState{
		io:        io,
		Target:    Order{-1, elevio.Cab},
		CurrFloor: initFloor,
		PrevFloor: -1,
		Dir:       elevio.Stop,
		Behavior:  Idle,
		Orders:    orders,
	}
}

func (e *ElevState) String() string {
	return fmt.Sprintf("{ Target: %+v, CurrFloor: %d, PrevFloor: %d, Dir: %v, Behavior: %s, Orders: %+v }",
		e.Target, e.CurrFloor, e.PrevFloor, e.Dir, e.Behavior, e.Orders)
}

// ---- Event Handlers ----//

func (e *ElevState) OnInitBetweenFloors() {
	fmt.Println("Initializing: Between floors")
	e.SetDir(elevio.Down)
	e.Behavior = Moving
}

func (e *ElevState) OnOrderRequest(order elevio.ButtonEvent) {
	fmt.Printf("[ORDER] %+v\n", order)
	e.io.SetButtonLamp(order.Button, order.Floor, true)
	e.Orders[order.Floor][order.Button] = true
	switch e.Behavior {
	case Idle:
		// Mark as active

		// Set Target floor
		e.Target.RType = order.Button
		e.Target.Floor = order.Floor

		e.Dir, e.Behavior = ChooseDirection(e)

		e.io.SetMotorDirection(e.Dir)

	case Moving:
	case DoorOpen:
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
	case Moving:
		if ShouldStop(e) {
			// stop
			e.Dir = elevio.Stop
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
		e.io.SetMotorDirection(elevio.Stop)
	} else {
		e.io.SetMotorDirection(e.Dir)
	}

	fmt.Printf("State: %v\n", e)
}

func (e *ElevState) OnStopSignal(stop bool) {
	fmt.Printf("[STOP] %+v\n", stop)
	for f := range e.Orders {
		for b := range e.Orders[f] {
			e.io.SetButtonLamp(elevio.ButtonType(b), f, false)
		}
	}

	fmt.Printf("State: %v\n", e)
}

func (e *ElevState) SetAllLights() {
	for f := range e.Orders {
		for b := range e.Orders[f] {
			fmt.Printf("F: %v, B: %v, on: %v ", elevio.ButtonType(b), f, e.Orders[f][b])
			e.io.SetButtonLamp(elevio.ButtonType(b), f, e.Orders[f][b])
		}
		fmt.Println()
	}
}

// <-ticker.C:
// 		switch elev.Behavior {
// 		case Idle:
// 			if elev.Target.Floor == -1 {
// 				// Check for any active orders
// 				found := false
// 				for f := 0; f < numFloors && !found; f++ {
// 					for b := elevio.ButtonType(0); b < 3 && !found; b++ {
// 						if elev.Orders[f][b] {
// 							elev.Target.Floor = f
// 							elev.Target.RType = b
// 							found = true
// 						}
// 					}
// 				}
//
// 				if found {
// 					fmt.Println("Target floor set to:", elev.Target.Floor)
// 					elevIoDriver.SetStopLamp(false)
//
// 					// Set initial direction when starting to move
// 					if elev.Target.Floor > elev.CurrFloor {
// 						elev.Dir = elevio.MD_Up
// 					} else if elev.Target.Floor < elev.CurrFloor {
// 						elev.Dir = elevio.MD_Down
// 					}
// 					elevIoDriver.SetMotorDirection(elev.Dir)
// 					elev.Behavior = Moving
// 				}
// 			}
// 		case Moving:
// 			tF := elev.Target.Floor
// 			cF := elev.CurrFloor
// 			if tF == cF {
// 				elev.Behavior = DoorOpen
// 				fmt.Printf("Arrived at floor: %v, curr=%v\n", tF, cF)
// 				continue
// 			}
//
// 			var newDir elevio.MotorDirection
// 			if tF > cF {
// 				newDir = elevio.MD_Up
// 			} else {
// 				newDir = elevio.MD_Down
// 			}
//
// 			// Only send motor direction if it changed
// 			if newDir != elev.Dir {
// 				elev.SetDir(newDir)
// 			}
//
// 			if elev.PrevFloor != cF {
// 				elev.PrevFloor = cF
// 				elev.Behavior = AtFloor
// 			}
// 		case AtFloor:
//
// 		case DoorOpen:
// 			fmt.Println("Door open at floor:", elev.CurrFloor)
// 			elevIoDriver.SetStopLamp(true)
// 			for b := range elev.Orders[elev.CurrFloor] {
// 				elev.Orders[elev.CurrFloor][b] = false
// 				elevIoDriver.SetButtonLamp(elevio.ButtonType(b), elev.CurrFloor, false)
// 			}
// 			elev.Target.Floor = -1
// 			elev.Target.RType = elevio.BT_Cab
// 			elev.SetDir(elevio.MD_Stop)
// 			elevIoDriver.SetDoorOpenLamp(true)
// 			fmt.Println("Now: ", time.Now())
// 			time.Sleep(3 * time.Second)
//
// 			fmt.Println("Three seconds passed: ", time.Now())
// 			elevIoDriver.SetDoorOpenLamp(false)
// 			elev.Behavior = Idle
// 			printState()
// 		}
// 	}
