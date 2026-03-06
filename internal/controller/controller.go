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

func OnFloorArrival(wv *statesync.Worldview, actionChan chan any) {
	remote := wv.GetRemoteElevator()
	// calls := Calls{
	// 	HallCalls: wv.GetAllHallCalls(),
	// 	CabCalls:  remote.CabCalls,
	// }
	// if remote.Behavior == elevator.BMoving {
	slog.Info("Should stop")
	actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
	actionChan <- elevator.DoorAction{Open: true}
	// ClearAtCurrentFloor(wv, remote, &calls)
	actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: remote.CurrentFloor, State: false}

	// time.Sleep(2 * time.Second)
	actionChan <- elevator.DoorAction{Open: false}
	slog.Info("Finished stopping")
	// }
}

func Start(wv *statesync.Worldview, trigger chan struct{}, actionChan chan any, newOrderChan chan statesync.Order, hcLightChan chan statesync.Order) {
	for {
		select {
		case order := <-hcLightChan:
			actionChan <- elevator.LightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: !order.Completed}
		case order := <-newOrderChan:
			actionChan <- elevator.LightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: !order.Completed}
		case <-trigger:
			slog.Warn("trigger!!!")
			local := wv.GetRemoteElevator()
			nearestHallCall, hallCallDirection := CalculateNearestHallCall(wv)
			nearestCabCall := CalculateNearestCabCall(wv)
			anyCallsAvailable := nearestHallCall != -1 || nearestCabCall != -1
			slog.Info("[Controller] Triggered order calculation", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "any?", anyCallsAvailable)

			if !anyCallsAvailable {
				slog.Info("[Controller] No calls available", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)
				actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
				continue
			}

			if nearestHallCall == local.CurrentFloor {
				err := wv.CompleteHallCall(nearestHallCall, hallCallDirection)
				if err != nil {
					slog.Error("[Controller] Got worldview error", "error", err)
				}
				slog.Info("[Controller] Completed HallCall", "floor", nearestHallCall, "direction", hallCallDirection)
			}

			if nearestCabCall == local.CurrentFloor {
				err := wv.SetCabCall(nearestCabCall, false)
				if err != nil {
					slog.Error("[Controller] Got worldview error", "error", err)
				}
				slog.Info("[Controller] Completed CabCall", "floor", nearestCabCall)
			}

			if nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor {
				slog.Info("[Controller] Arrived at order", "CabCall", nearestCabCall, "HallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				OnFloorArrival(wv, actionChan)
				continue
			}

			dir := elevio.MDStop
			isCabCAll := nearestCabCall != -1 && nearestHallCall == -1

			if isCabCAll {
				if nearestCabCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}
				slog.Info("[Controller] sending action for cabcall", "floor", nearestCabCall, "dir", dir, "currentDir", local.Direction)

				if dir == local.Direction || local.Direction == elevio.MDStop {
					actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
				}

			} else {
				if nearestHallCall < local.CurrentFloor {
					dir = elevio.MDDown
				} else {
					dir = elevio.MDUp
				}

				slog.Info("[Controller] sending action for hallcall", "floor", nearestHallCall, "dir", dir, "currentDir", local.Direction)
				if dir == local.Direction || local.Direction == elevio.MDStop {
					actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dir}
				}
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
	slog.Info("[CalculateNearestHallCall]", "Skip down?", skipDownCalls, "skip up?", skipUpCalls)

	for floor, hallCall := range hallCalls {
		for dirIdx := range hallCalls[floor] {
			dir := statesync.HallCallDir(dirIdx)
			if hallCall[dir].State == statesync.HSNone ||
				// skipDownCalls && dir == statesync.HDDown ||
				// skipUpCalls && dir == statesync.HDUp ||
				hallCall[dir].By != wv.LocalID {
				continue
			}
			slog.Info("[CalculateNearestHallCall] Found a valid hallcall", "By", hallCall[dir].By, "floor", floor)

			distance := int(math.Abs(float64(local.CurrentFloor) - float64(floor)))
			slog.Info("[CalculateNearestHallCall]", "distance", distance, "nearestCall", nearestCall, "distance < nearestCall", distance < nearestCall)
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
