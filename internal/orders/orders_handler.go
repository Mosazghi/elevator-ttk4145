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
	id    int
	floor int
	cost  int
}

type Penalty int

const (
	PenaltyObstructed     = 20
	PenaltyWrongDirection = 10
)

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
		nearestHallCall, hallCallDirection := CalculateNearestHallCall(&wv)
		nearestCabCall := CalculateNearestCabCall(&wv)
		slog.Info("[GetNextOrder]", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

		local := wv.GetRemoteElevator()

		if nearestHallCall == local.CurrentFloor {
			err := wv.CompleteHallCall(nearestHallCall, hallCallDirection)
			if err != nil {
				slog.Error("[GetNextOrder] Got worldview error", "error", err)
			}
			slog.Info("[GetNextOrder] Completed HallCall", "floor", nearestHallCall)
		}

		if nearestCabCall == local.CurrentFloor {
			err := wv.SetCabCall(nearestCabCall, false)
			if err != nil {
				slog.Error("[GetNextOrder] Got worldview error", "error", err)
			}
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
				err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
				if err != nil {
					slog.Error("[RunCost] Got worldview error", "error", err)
				}
				slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir)
			} else {
				slog.Info("[RunCost] Not processing", "winner id", winner.id, "local id", wv.LocalID)
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

func CalculateNearestHallCall(wv *statesync.Worldview) (int, statesync.HallCallDir) {
	slog.Info("[CalculateNearestHallCall] Starting")
	local := wv.GetRemoteElevator()
	hallCalls := wv.GetAllHallCalls()
	nearestCall := wv.NumFloors + 1
	direction := statesync.HDDown

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
				direction = statesync.HallCallDir(dir)
				slog.Info("[CalculateNearestHallCall] New nearest hallcall", "floor", floor)
			}
		}
	}

	return nearestCall, direction
}

func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}
	slog.Info("[CalculateCost] Starting")
	currentElevatorCost := ElevatorCost{-1, wv.NumFloors + 1, 0}

	for id, elev := range wv.ElevatorStates {
		isObstructed := elev.Behavior == elevator.BObstructed
		currentElevatorCost.id = id
		currentElevatorCost.floor = 0
		currentElevatorCost.cost = 0

		if isObstructed {
			slog.Info("[CalculateCost] Elevator is obstructed")
			currentElevatorCost.cost += PenaltyObstructed
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.cost += PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.cost += PenaltyWrongDirection
		}

		cost := int(math.Abs(float64(currentElevatorCost.cost) - float64(floor)))
		currentElevatorCost.floor = floor
		slog.Info("[CalculateCost] found valid elevator", "cost", cost, "floor", floor, "id", currentElevatorCost.id)

		if cost < winner.cost {
			winner.id = currentElevatorCost.id
			winner.cost = cost
			winner.floor = currentElevatorCost.floor
			slog.Info("[CalculateCost] Got new winner", "id", winner.id, "cost", winner.cost, "floor", winner.floor)
		}
	}
	return winner
}
