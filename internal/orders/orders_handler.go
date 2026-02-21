package orders

import (
	"log/slog"
	"math"
	"time"

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

type Calls struct {
	HallCalls [][2]statesync.HallCallPairState
	CabCalls  []bool
}

func FSM(floorTrigger chan statesync.Worldview, actionChan chan any) {
	for wv := range floorTrigger {
		remote := wv.GetRemoteElevator()
		calls := Calls{
			HallCalls: wv.GetAllHallCalls(),
			CabCalls:  remote.CabCalls,
		}
		if ShouldStop(&remote, calls) {
			actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
			actionChan <- elevator.DoorAction{Open: true}

			time.Sleep(2 * time.Second)
			actionChan <- elevator.DoorAction{Open: false}

			slog.Info("Should stop")
		}
	}
}

func GetNextAction(wv *statesync.Worldview, trigger chan struct{}, actionChan chan any) {
	for range trigger {
		nearestHallCall, hallCallDirection := CalculateNearestHallCall(wv)
		nearestCabCall := CalculateNearestCabCall(wv)

		slog.Info("[GetNextOrder] ", "nearestCabCall", nearestCabCall, "nearestHallCall", nearestHallCall)

		if nearestHallCall == -1 && nearestCabCall == -1 {
			continue
		}

		local := wv.GetRemoteElevator()

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
			actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
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

// hallDirToButtonType maps statesync HallCallDir to the correct elevio ButtonType.
// statesync: HDDown=0, HDUp=1 | elevio: HallUp=0, HallDown=1
func hallDirToButtonType(dir statesync.HallCallDir) elevio.ButtonType {
	if dir == statesync.HDDown {
		return elevio.HallDown
	}
	return elevio.HallUp
}

func RunCost(wvChan chan statesync.Worldview, trigger chan struct{}, actionChan chan any) {
	prevHC := make(map[int][2]statesync.HallCallPairState)

	for wv := range wvChan {
		hallCalls := wv.GetAllHallCalls()

		// Propagate light changes driven by any hall call state transition
		// (e.g. a new order arriving from another node).
		for floor := range hallCalls {
			for d := range hallCalls[floor] {
				dir := statesync.HallCallDir(d)
				prevState := prevHC[floor][dir]
				currentState := hallCalls[floor][dir]

				if prevState.State != currentState.State {
					btn := hallDirToButtonType(dir)
					switch currentState.State {
					case statesync.HSNone:
						actionChan <- elevator.LightAction{ButtonType: btn, Floor: floor, State: elevator.LSOff}
					case statesync.HSAvailable, statesync.HSProcessing:
						actionChan <- elevator.LightAction{ButtonType: btn, Floor: floor, State: elevator.LSOn}
					}
				}
				temp := prevHC[floor]
				temp[dir] = currentState
				prevHC[floor] = temp
			}
		}

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				if hallCall[dir].State == statesync.HSNone {
					continue
				}

				isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= len(wv.ElevatorStates)
				isAvailable := hallCall[dir].State == statesync.HSAvailable

				if hallCall[dir].State == statesync.HSProcessing && hallCall[dir].By == wv.LocalID {
					trigger <- struct{}{}
				}

				if !isConfirmedByAll || !isAvailable {
					slog.Info("[RunCost]", "confirmed by all", isConfirmedByAll, "available", isAvailable)
					continue
				}

				// slog.Info("[RunCost] Calculating cost")
				winner := CalculateCost(&wv, floor, statesync.HallCallDir(dir))
				if winner.id == wv.LocalID {
					err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[RunCost] Got worldview error", "error", err)
					}
					slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir, "id", winner.id)
				}
				slog.Warn("[RunCost] Order picked up!")
				actionChan <- elevator.LightAction{ButtonType: hallDirToButtonType(statesync.HallCallDir(dir)), Floor: floor, State: elevator.LSOn}
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

func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}
	// slog.Info("[CalculateCost] Starting")

	for id, elev := range wv.ElevatorStates {
		currentElevatorCost := ElevatorCost{-1, wv.NumFloors + 1, 0}
		isObstructed := elev.Behavior == elevator.BObstructed
		currentElevatorCost.id = id

		if isObstructed {
			// slog.Info("[CalculateCost] Elevator is obstructed")
			currentElevatorCost.cost += PenaltyObstructed
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			// slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.cost += PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			// slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.cost += PenaltyWrongDirection
		}

		distance := int(math.Abs(float64(elev.CurrentFloor - floor)))

		currentElevatorCost.cost += distance

		currentElevatorCost.cost += id

		// slog.Info("[CalculateCost] found valid elevator", "cost", cost, "floor", floor, "id", currentElevatorCost.id)

		if currentElevatorCost.cost < winner.cost {
			winner.id = currentElevatorCost.id
			winner.cost = currentElevatorCost.cost
			winner.floor = currentElevatorCost.floor
			slog.Info("[CalculateCost] Got new winner", "id", winner.id, "cost", winner.cost, "floor", winner.floor)
		}
	}
	return winner
}
