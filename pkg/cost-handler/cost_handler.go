package costhandler

// import (
// 	. "Heisern/pkg/elevator"
// )

// func FindBestCost(e *ElevState) (Order, bool) {
// 	currDir := e.Dir
// 	target := e.Target
// 	cF := e.CurrFloor
// 	orders := e.Orders
// 	// var bestOrder Order
//
// 	for f := range orders {
// 		for b := range orders[f] {
// 			order := orders[f][b]
// 			if order {
// 				// if found order's floor is below ours AND we are moving up, then skip
// 				if cF > f && currDir == elevio.MD_Up {
// 					continue
// 				}
// 				// If order found on our floor then skip
// 				if cF == f {
// 					continue
// 				}
// 				// If order's floor is larger than targets floor then skip
// 				if f > target.Floor && currDir == elevio.MD_Up {
// 					continue
// 				}
//
// 				// Same for down
// 				if f < target.Floor && currDir == elevio.MD_Down {
// 					continue
// 				}
//
// 				switch elevio.ButtonType(b) {
// 				case elevio.BT_HallUp:
// 					if currDir == elevio.MD_Up {
// 						return Order{
// 							Floor: f,
// 							RType: elevio.BT_HallUp,
// 						}, true
// 					}
//
// 				case elevio.BT_HallDown:
//
// 					if currDir == elevio.MD_Down {
// 						return Order{
// 							f,
// 							elevio.BT_HallDown,
// 						}, true
// 					}
// 				case elevio.BT_Cab:
//
// 				}
// 				// If current floor and there is an order that is right above us and is
// 				// different from currentTarget
// 			}
// 		}
// 	}
//
// 	return Order{
// 		-1,
// 		elevio.BT_HallUp,
// 	}, true
// }
