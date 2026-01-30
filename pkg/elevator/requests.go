package elevator

import (
	"fmt"

	"Heisern/pkg/elevio"
)

func HasOrders(e *ElevState) bool {
	for f := range e.Orders {
		for b := range e.Orders[f] {
			if e.Orders[f][b] {
				return true
			}
		}
	}
	return false
}

func HasOrdersAbove(e *ElevState) bool {
	for f := e.CurrFloor + 1; f < len(e.Orders); f++ {
		for b := range e.Orders[f] {
			if e.Orders[f][b] {
				return true
			}
		}
	}
	return false
}

func HasOrdersBelow(e *ElevState) bool {
	for f := 0; f < e.CurrFloor; f++ {
		for b := range e.Orders[f] {
			if e.Orders[f][b] {
				return true
			}
		}
	}
	return false
}

func ShouldStop(e *ElevState) bool {
	switch e.Dir {
	case elevio.Down:
		return e.Orders[e.CurrFloor][elevio.HallDown] ||
			e.Orders[e.CurrFloor][elevio.Cab] ||
			!HasOrdersBelow(e) // Reached "dead-end"
	case elevio.Up:

		return e.Orders[e.CurrFloor][elevio.HallUp] ||
			e.Orders[e.CurrFloor][elevio.Cab] ||
			!HasOrdersAbove(e) // Reached "dead-end"
	case elevio.Stop:
		return true
	}

	return true
}

func ChooseDirection(e *ElevState) (elevio.MotorDirection, Behavior) {
	switch e.Dir {
	case elevio.Up:
	case elevio.Down:
	case elevio.Stop:
		if HasOrdersAbove(e) {
			return elevio.Up, Moving
		}

		if HasOrdersBelow(e) {
			return elevio.Down, Moving
		}
	}
	return elevio.Stop, Idle
}

func ClearAtCurrentFloor(e *ElevState) {
	e.Orders[e.CurrFloor][elevio.Cab] = false

	if e.Orders[e.CurrFloor][elevio.HallUp] {
		e.Orders[e.CurrFloor][elevio.HallUp] = false
	}

	if e.Orders[e.CurrFloor][elevio.HallDown] {
		e.Orders[e.CurrFloor][elevio.HallDown] = false
	}
}

func PrintOrders(e *ElevState) {
	for f := range e.Orders {
		for b := range e.Orders[f] {
			if e.Orders[f][b] {
				fmt.Printf("Order at floor %d, button %v\n", f, elevio.ButtonType(b))
			}
		}
	}
}
