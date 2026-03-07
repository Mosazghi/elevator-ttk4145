package controller

import (
	"log/slog"
	"math"
	"time"

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
	slog.Debug("Should stop")
	actionChan <- elevator.MoveAction{Behavior: elevator.BDoorOpen, Direction: elevio.MDStop}
	actionChan <- elevator.DoorAction{Open: true}
	actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: remote.CurrentFloor, State: false}

	time.AfterFunc(3*time.Second, func() {
		actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
		actionChan <- elevator.DoorAction{Open: false}
		slog.Debug("Finished stopping")
	})
}

func Start(wv *statesync.Worldview, trigger chan struct{}, actionChan chan any, newOrderChan chan statesync.Order, hcLightChan chan statesync.Order) {
	prev := -1
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
				slog.Debug("[Controller] No calls available", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)
				actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
				continue
			}

			allowedToProcess := prev != local.CurrentFloor || local.Behavior == elevator.BIdle
			isCabCAll := nearestCabCall != -1 && nearestHallCall == -1

			slog.Warn("[Controller]", "previousFloor", prev, "currentFloor", local.CurrentFloor)
			slog.Info("[Controller]", "allowedToProcess", allowedToProcess, "isCabCAll", isCabCAll)

			if !allowedToProcess {
				continue
			}

			arrivalOk := nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor

			slog.Info("[Controller]", "arrvalOk", arrivalOk)

			if arrivalOk {
				if nearestCabCall == local.CurrentFloor {
					wv.SetCabCall(nearestCabCall, false)
				}

				if nearestHallCall == local.CurrentFloor {
					wv.CompleteHallCall(nearestHallCall, hallCallDirection)
				}

				OnFloorArrival(wv, actionChan)
				prev = local.CurrentFloor

			} else {
				// We only allow setting a new direction if we are idle. i.e after we have stopped
				if local.Behavior != elevator.BIdle { // Refactor for clearity needed
					continue
				}

				if isCabCAll {
					actionChan <- DetermineAction(nearestCabCall, local)
				} else {
					actionChan <- DetermineAction(nearestHallCall, local)
				}
			}
		}
	}
}

func DetermineAction(callFloor int, localElevator statesync.RemoteElevatorState) elevator.MoveAction {
	var direction elevio.MotorDirection

	if callFloor < localElevator.CurrentFloor {
		direction = elevio.MDDown
	} else {
		direction = elevio.MDUp
	}

	return elevator.MoveAction{Behavior: elevator.BMoving, Direction: direction}
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
		// slog.Debug("[CalculateNearestCabCall]", "cost", cost, "floor", floor)

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
	// slog.Debug("[CalculateNearestCabCall]", "bestFloor", bestFloor)
	return bestFloor
}

func CalculateNearestHallCall(wv *statesync.Worldview) (int, statesync.HallCallDir) {
	localElevator := wv.GetRemoteElevator()
	hallCalls := wv.GetAllHallCalls()
	closestFloorDirection := statesync.HDDown
	bestCost := math.MaxInt
	clostestFloor := -1

	for floor, hallCall := range hallCalls {
		for direction := range hallCalls[floor] {
			if hallCall[direction].By != localElevator.ID ||
				hallCall[direction].State == statesync.HSNone {
				continue
			}

			cost := int(math.Abs(float64(floor - localElevator.CurrentFloor)))

			wrongDirection := (localElevator.Direction == elevio.MDDown && floor > localElevator.CurrentFloor) ||
				(localElevator.Direction == elevio.MDUp && floor < localElevator.CurrentFloor)

			if wrongDirection {
				cost += orders.PenaltyWrongDirection
			}

			if cost < bestCost {
				bestCost = cost
				clostestFloor = floor
				closestFloorDirection = statesync.HallCallDir(direction)
			}
		}
	}

	slog.Warn("[CalculateNearestCabCall]", "clostestFloor", clostestFloor, "closestFloorDirection", closestFloorDirection)

	return clostestFloor, closestFloorDirection
}
