package elevator

// import (
// 	"fmt"

// 	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
// )

// func HasOrders(e *ElevatorState) bool {
// 	for f := range e.Orders {
// 		for b := range e.Orders[f] {
// 			if e.Orders[f][b] {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

// func HasOrdersAbove(e *ElevState) bool {
// 	for f := e.CurrFloor + 1; f < len(e.Orders); f++ {
// 		for b := range e.Orders[f] {
// 			if e.Orders[f][b] {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

// func HasOrdersBelow(e *ElevState) bool {
// 	for f := 0; f < e.CurrFloor; f++ {
// 		for b := range e.Orders[f] {
// 			if e.Orders[f][b] {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

// func ShouldStop(e *ElevState) bool {
// 	switch e.Dir {
// 	case elevio.MDDown:
// 		return e.Orders[e.CurrFloor][elevio.HallDown] ||
// 			e.Orders[e.CurrFloor][elevio.Cab] ||
// 			!HasOrdersBelow(e) // Reached "dead-end"
// 	case elevio.MDUp:

// 		return e.Orders[e.CurrFloor][elevio.HallUp] ||
// 			e.Orders[e.CurrFloor][elevio.Cab] ||
// 			!HasOrdersAbove(e) // Reached "dead-end"
// 	case elevio.MDStop:
// 		return true
// 	}

// 	return true
// }

// func ChooseDirection(e *ElevState) (elevio.MotorDirection, Behavior) {
// 	switch e.Dir {
// 	case elevio.MDUp:
// 	case elevio.MDDown:
// 	case elevio.MDStop:
// 		if HasOrdersAbove(e) {
// 			return elevio.MDUp, BMoving
// 		}

// 		if HasOrdersBelow(e) {
// 			return elevio.MDDown, BMoving
// 		}
// 	}
// 	return elevio.MDStop, BIdle
// }

// func ClearAtCurrentFloor(e *ElevState) {
// 	e.Orders[e.CurrFloor][elevio.Cab] = false

// 	if e.Orders[e.CurrFloor][elevio.HallUp] {
// 		e.Orders[e.CurrFloor][elevio.HallUp] = false
// 	}

// 	if e.Orders[e.CurrFloor][elevio.HallDown] {
// 		e.Orders[e.CurrFloor][elevio.HallDown] = false
// 	}
// }

// func PrintOrders(e *ElevState) {
// 	for f := range e.Orders {
// 		for b := range e.Orders[f] {
// 			if e.Orders[f][b] {
// 				fmt.Printf("Order at floor %d, button %v\n", f, elevio.ButtonType(b))
// 			}
// 		}
// 	}
// }
