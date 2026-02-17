package orders

import (
	"log/slog"
	"math"
	"slices"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type ElevatorCost struct {
	id       int
	distance int
}

// 1. Update the worldview
// 2. Calculate the nearestcab and hallcall
// 3. If any of them are the same as our current pos -> stop || Go that said direction

func GetNextOrder(wvChan chan statesync.Worldview, actionChan chan elevator.Action) {
	for {
		select {
		case wv := <-wvChan:
			slog.Debug("[GetNextOrder] Got new worldview")
			RunCost(&wv)
			nearestHallCall := CalculateNearestHallCall(&wv)
			nearestCabCall := CalculateNearestCabCall(&wv)
			slog.Debug("[GetNextOrder]", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

			local := wv.GetRemoteElevator()

			if nearestHallCall == local.CurrentFloor {
				wv.CompleteHallCall(nearestHallCall, statesync.HDNone)
				slog.Debug("[GetNextOrder] Completed HallCall", "floor", nearestHallCall)
			}

			if nearestCabCall == local.CurrentFloor {
				wv.SetCabCall(nearestCabCall, false)
				slog.Debug("[GetNextOrder] Completed CabCall", "floor", nearestCabCall)
			}

			if nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor {
				slog.Debug("[GetNextOrder] Arrived at order", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				actionChan <- elevator.Action{Behavior: elevator.BIdle, Direction: elevio.MDStop}
				continue
			}

			if nearestCabCall < local.CurrentFloor || nearestHallCall < local.CurrentFloor {
				slog.Debug("[GetNextOrder] Moving down", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				actionChan <- elevator.Action{Behavior: elevator.BMoving, Direction: elevio.MDDown}
			} else {
				slog.Debug("[GetNextOrder] Moving up", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
				actionChan <- elevator.Action{Behavior: elevator.BMoving, Direction: elevio.MDUp}
			}
		}
	}
}

func RunCost(wv *statesync.Worldview) {
	hallCalls := wv.GetAllHallCalls()

	for floor, hallCall := range hallCalls {
		for dir := range 2 {
			if hallCall[dir].State == statesync.HSNone {
				continue
			}

			isConfirmedByAll := len(hallCall[dir].ConfirmedBy) == len(wv.ElevatorStates)
			isNotSet := hallCall[dir].State == statesync.HSAvailable

			if !isConfirmedByAll && !isNotSet {
				continue
			}

			slog.Debug("[RunCost] Calculating cost")
			winner := CalculateCost(wv, floor, dir)
			if winner.id == wv.LocalID {
				wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
				slog.Debug("[RunCost] Set to processing", "floor", floor, "Direction", dir)
			}
		}
	}
}

func CalculateNearestCabCall(wv *statesync.Worldview) int {
	// Localy find nearest cab call
	localElevator := wv.GetRemoteElevator()
	cabCalls := localElevator.CabCalls
	reversed := false
	nearestCall := 0

	if localElevator.Direction == elevio.MDDown {
		slices.Reverse(cabCalls)
		reversed = true
		slog.Debug("[CalculateNearestCabCall] Reversed cabCalls (got call downwards)")
	}

	for floor, isSetcabCall := range cabCalls {
		if !isSetcabCall {
			continue
		}

		if reversed {
			nearestCall = localElevator.NumFloors - floor
		} else {
			nearestCall = floor
		}

		slog.Debug("[CalculateNearestCabCall] Got nearestCabCall", "floor", nearestCall)
		break
	}

	return nearestCall
}

func CalculateNearestHallCall(wv *statesync.Worldview) int {
	local := wv.GetRemoteElevator()
	hallCalls := wv.GetAllHallCalls()
	nearestCall := wv.NumFloors + 1

	skipDownCalls := local.Direction != elevio.MDDown && local.Direction != elevio.MDStop
	skipUpCalls := local.Direction != elevio.MDUp && local.Direction != elevio.MDStop

	for floor, hallCall := range hallCalls {
		for dir := range 2 {
			if hallCall[dir].State == statesync.HSNone || skipDownCalls && dir == 0 || skipUpCalls && dir == 1 {
				continue
			}

			distance := int(math.Abs(float64(local.CurrentFloor) - float64(floor)))
			if hallCall[dir].By == wv.LocalID && distance < nearestCall {
				nearestCall = floor
			}
		}
	}

	return nearestCall
}

// TODO: EDGE CASE: two or more elevators at same spot/floor. lowest id wins!
// Theory it will automaticlay do this idea since we count from 0->largest id?
func CalculateCost(wv *statesync.Worldview, floor int, dir int) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1}
	for id, elev := range wv.ElevatorStates {
		isObstructed := elev.Behavior == elevator.BObstructed

		if isObstructed {
			slog.Debug("[CalculateCost] Elevator is obstructed")
			continue
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			slog.Debug("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			continue
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			slog.Debug("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			continue
		}

		distance := int(math.Abs(float64(elev.CurrentFloor) - float64(floor)))

		if distance < winner.distance {
			winner.id = id
			winner.distance = distance
			slog.Debug("[CalculateCost] Got new winner", "id", winner.id, "distance", winner.distance)
		}
	}
	return winner
}
