package controller

import (
	"log/slog"
	"math"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/orders"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type Calls struct {
	HallCalls [][2]statesync.HallCallPairState
	CabCalls  []bool
}

func Start(wv *statesync.Worldview, trigger chan struct{}, actionChan chan any, newOrderChan chan statesync.Order) {
	for {
		select {
		case order := <-newOrderChan:
			actionChan <- elevator.SingleLightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: order.Completed}
		case <-trigger:
			local := wv.GetRemoteElevator()

			slog.Warn("trigger!!!")

			nearestHallCall, hallCallDirection := CalculateNearestHallCall(wv)
			nearestCabCall := CalculateNearestCabCall(wv)

			if nearestHallCall == -1 && nearestCabCall == -1 {
				slog.Info("[GetNextOrder]", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)
				actionChan <- elevator.StopAction{}
				continue
			}

			slog.Info("[GetNextOrder] ", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

			if nearestHallCall == local.CurrentFloor {
				err := wv.CompleteHallCall(nearestHallCall, hallCallDirection)
				if err != nil {
					slog.Error("[GetNextOrder] Got worldview error", "error", err)
				}
				slog.Info("[GetNextOrder] Completed HallCall", "floor", nearestHallCall, "direction", hallCallDirection)
			}

			if nearestCabCall == local.CurrentFloor {
				err := wv.SetCabCall(nearestCabCall, false)
				if err != nil {
					slog.Error("[GetNextOrder] Got worldview error", "error", err)
				}
				slog.Info("[GetNextOrder] Completed CabCall", "floor", nearestCabCall)
			}

			if nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor {
				slog.Info("[GetNextOrder] Arrived at order", "CabCall", nearestCabCall, "HallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				trigger <- struct{}{}
				continue
			}

			dir := elevio.MDStop

			if nearestCabCall != -1 && nearestHallCall == -1 {
				if nearestCabCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}
				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
			} else {
				// slog.Info("[GetNextOrder] sending action for hallcall", "floor", nearestHallCall)
				if nearestHallCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}
				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
			}
		}
	}
}

func Start_(wv *statesync.Worldview, trigger chan struct{}, actionChan chan any, newOrderChan chan statesync.Order) {
	for {
		select {
		case order := <-newOrderChan:
			actionChan <- elevator.SingleLightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: order.Completed}

		case <-trigger:
			slog.Warn("trigger!!!")
			local := wv.GetRemoteElevator()

			// If already moving, don't re-issue movement commands — OnFloorArrival handles stopping.
			// Re-issuing causes direction oscillation when multiple orders are pending.
			if local.Behavior == elevator.BMoving {
				continue
			}

			nearestHallCall, _ := CalculateNearestHallCall(wv)
			nearestCabCall := CalculateNearestCabCall(wv)

			// slog.Info("[GetNextOrder] ", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

			if nearestHallCall == -1 && nearestCabCall == -1 {
				continue
			}

			// if nearestHallCall == local.CurrentFloor {
			// 	err := wv.CompleteHallCall(nearestHallCall, hallCallDirection)
			// 	if err != nil {
			// 		slog.Error("[GetNextOrder] Got worldview error", "error", err)
			// 	}
			// 	slog.Info("[GetNextOrder] Completed HallCall", "floor", nearestHallCall, "direction", hallCallDirection)
			// }

			// if nearestCabCall == local.CurrentFloor {
			// 	err := wv.SetCabCall(nearestCabCall, false)
			// 	if err != nil {
			// 		slog.Error("[GetNextOrder] Got worldview error", "error", err)
			// 	}
			// 	slog.Info("[GetNextOrder] Completed CabCall", "floor", nearestCabCall)
			// }

			if nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor {
				// slog.Info("[GetNextOrder] Arrived at order", "CabCall", nearestCabCall, "HallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				// actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
				continue
			}

			dir := elevio.MDStop

			if nearestCabCall != -1 && nearestHallCall == -1 {
				if nearestCabCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}
				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
			} else {
				// slog.Info("[GetNextOrder] sending action for hallcall", "floor", nearestHallCall)
				if nearestHallCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}
				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
			}
		}
	}
}

// CalculateNearestCabCall finds closest cab call based on current local elevator position
func CalculateNearestCabCall(wv *statesync.Worldview) int {
	local := wv.GetRemoteElevator()
	bestFloor := -1
	bestCost := math.MaxInt

	for floor, set := range local.CabCalls {
		if !set {
			continue
		}

		cost := int(math.Abs(float64(floor - local.CurrentFloor)))
		// slog.Info("[CalculateNearestCabCall]", "cost", cost, "floor", floor)

		// optional: wrong-direction penalty
		if (local.Direction == elevio.MDDown && floor > local.CurrentFloor) ||
			(local.Direction == elevio.MDUp && floor < local.CurrentFloor) {
			cost += 10
		}

		if cost < bestCost {
			bestCost = cost
			bestFloor = floor
		}
	}
	// slog.Info("[CalculateNearestCabCall]", "bestFloor", bestFloor)
	return bestFloor
}

func CalculateNearestHallCall(wv *statesync.Worldview) (int, statesync.HallCallDir) {
	// slog.Info("[CalculateNearestHallCall] Starting")
	local := wv.GetRemoteElevator()
	hallCalls := wv.GetAllHallCalls()
	direction := statesync.HDDown
	nearestCall := -1

	skipDownCalls := local.Direction != elevio.MDDown && local.Direction != elevio.MDStop
	skipUpCalls := local.Direction != elevio.MDUp && local.Direction != elevio.MDStop
	// slog.Info("[CalculateNearestHallCall]", "Skip down?", skipDownCalls, "skip up?", skipUpCalls)

	for floor, hallCall := range hallCalls {
		for dirIdx := range hallCalls[floor] {
			dir := statesync.HallCallDir(dirIdx)
			if hallCall[dir].State == statesync.HSNone ||
				skipDownCalls && dir == statesync.HDDown ||
				skipUpCalls && dir == statesync.HDUp ||
				hallCall[dir].By != wv.LocalID {
				continue
			}
			// slog.Info("[CalculateNearestHallCall] Found a valid hallcall", "By", hallCall[dir].By, "floor", floor)

			distance := int(math.Abs(float64(local.CurrentFloor) - float64(floor)))
			// slog.Info("[CalculateNearestHallCall]", "distance", distance, "nearestCall", nearestCall, "distance < nearestCall", distance < nearestCall)
			if distance < nearestCall || nearestCall == -1 {
				// slog.Info("[CalculateNearestHallCall]", "distance", distance, "id", wv.LocalID, "hallcallId", hallCall[dir].By)
				if hallCall[dir].By == wv.LocalID {
					nearestCall = floor
					direction = statesync.HallCallDir(dir)
					// slog.Info("[CalculateNearestHallCall] New nearest hallcall", "floor", floor)
				}
			}
		}
	}

	return nearestCall, direction
}
