package orders

import (
	"fmt"
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
//
//FIXME: Add error handling in '_ =' expressions
//
// Use channels as error handling? channel dead = Kill program, check in deafult state in statemachine

func GetNextOrder(wvChan chan statesync.Worldview, actionChan chan elevator.Action) {
	for wv := range wvChan {
		slog.Info("[GetNextOrder] Got new worldview")
		RunCost(&wv)
		nearestHallCall := CalculateNearestHallCall(&wv)
		nearestCabCall := CalculateNearestCabCall(&wv)
		slog.Info("[GetNextOrder]", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

		local := wv.GetRemoteElevator()

		if nearestHallCall == local.CurrentFloor {
			// FIXME: Fix the directions!!
			wv.CompleteHallCall(nearestHallCall, statesync.HDUp)
			wv.CompleteHallCall(nearestHallCall, statesync.HDDown)
			slog.Info("[GetNextOrder] Completed HallCall", "floor", nearestHallCall)
		}

		if nearestCabCall == local.CurrentFloor {
			wv.SetCabCall(nearestCabCall, false)
			slog.Info("[GetNextOrder] Completed CabCall", "floor", nearestCabCall)
		}

		if nearestCabCall == local.CurrentFloor || nearestHallCall == local.CurrentFloor {
			slog.Info("[GetNextOrder] Arrived at order", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
			actionChan <- elevator.Action{Behavior: elevator.BIdle, Direction: elevio.MDStop}
			continue
		}

		if nearestCabCall < local.CurrentFloor || nearestHallCall < local.CurrentFloor {
			slog.Info("[GetNextOrder] Moving down", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
			actionChan <- elevator.Action{Behavior: elevator.BMoving, Direction: elevio.MDDown}
		} else {
			slog.Info("[GetNextOrder] Moving up", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall, "currentPos", local.CurrentFloor)
			actionChan <- elevator.Action{Behavior: elevator.BMoving, Direction: elevio.MDUp}
		}
	}
}

func RunCost(wv *statesync.Worldview) {
	hallCalls := wv.GetAllHallCalls()

	for floor, hallCall := range hallCalls {
		for dir := range hallCalls[floor] {
			if hallCall[dir].State == statesync.HSNone {
				continue
			}

			isConfirmedByAll := len(hallCall[dir].ConfirmedBy) == len(wv.ElevatorStates)
			isNotSet := hallCall[dir].State == statesync.HSAvailable

			if !isConfirmedByAll || !isNotSet {
				continue
			}

			slog.Info("[RunCost] Calculating cost")
			winner := CalculateCost(wv, floor, statesync.HallCallDir(dir))
			if winner.id == wv.LocalID {
				wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
				slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir)
			} else {
				fmt.Printf("NOT PROCESSING!!\n")
			}
		}
	}
}

// CalculateNearestCabCall finds closest cab call based on current local elevator position
func CalculateNearestCabCall(wv *statesync.Worldview) int {
	slog.Info("[CalculateNearestCabCall] Starting")
	localElevator := wv.GetRemoteElevator()
	nearestCall := localElevator.NumFloors + 1
	cabCalls := localElevator.CabCalls
	startPosition := 0
	reversed := false

	if localElevator.Direction == elevio.MDDown {
		slices.Reverse(cabCalls)
		reversed = true
		startPosition = localElevator.NumFloors - localElevator.CurrentFloor
		slog.Info("[CalculateNearestCabCall] Reversed cabCalls (got call downwards)", "startPosition", startPosition)
	} else {
		startPosition = localElevator.CurrentFloor
		slog.Info("[CalculateNearestCabCall] Not reversed (got call upwards)", "startPosition", startPosition)
	}

	for floor, isSetcabCall := range cabCalls[startPosition:] {
		if !isSetcabCall {
			continue
		}

		if reversed {
			nearestCall = localElevator.NumFloors - (floor + startPosition)
		} else {
			nearestCall = floor + startPosition
		}

		slog.Info("[CalculateNearestCabCall] Found a valid cabcall", "floor", nearestCall)
		break
	}

	return nearestCall
}

func CalculateNearestHallCall(wv *statesync.Worldview) int {
	slog.Info("[CalculateNearestHallCall] Starting")
	local := wv.GetRemoteElevator()
	hallCalls := wv.GetAllHallCalls()
	nearestCall := wv.NumFloors + 1

	skipDownCalls := local.Direction != elevio.MDDown && local.Direction != elevio.MDStop
	skipUpCalls := local.Direction != elevio.MDUp && local.Direction != elevio.MDStop
	slog.Info("[CalculateNearestHallCall]", "Skip down?", skipDownCalls, "skip up?", skipUpCalls)

	for floor, hallCall := range hallCalls {
		for dir := range hallCalls[floor] {
			if hallCall[dir].State == statesync.HSNone || skipDownCalls && dir == 0 || skipUpCalls && dir == 1 {
				continue
			}
			slog.Info("[CalculateNearestHallCall] Found a valid hallcall", "By", hallCall[dir].By, "floor", floor)

			distance := int(math.Abs(float64(local.CurrentFloor) - float64(floor)))
			if hallCall[dir].By == wv.LocalID && distance < nearestCall {
				nearestCall = floor
				slog.Info("[CalculateNearestHallCall] New nearest hallcall", "floor", floor)
			}
		}
	}

	return nearestCall
}

// TODO: EDGE CASE: two or more elevators at same spot/floor. lowest id wins!
// Theory it will automaticlay do this idea since we count from 0->largest id?
func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1}
	for id, elev := range wv.ElevatorStates {
		isObstructed := elev.Behavior == elevator.BObstructed

		if isObstructed {
			slog.Info("[CalculateCost] Elevator is obstructed")
			continue
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			continue
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			continue
		}

		distance := int(math.Abs(float64(elev.CurrentFloor) - float64(floor)))

		if distance < winner.distance {
			winner.id = id
			winner.distance = distance
			slog.Info("[CalculateCost] Got new winner", "id", winner.id, "distance", winner.distance)
		}
	}
	return winner
}
