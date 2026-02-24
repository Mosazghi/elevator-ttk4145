package orders

import (
	"log/slog"
	"math"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type ElevatorCost struct {
	id    int
	floor int
	cost  int
}

// HallDirToButtonType maps statesync HallCallDir to the correct elevio ButtonType.
func HallDirToButtonType(dir statesync.HallCallDir) elevio.ButtonType {
	if dir == statesync.HDDown {
		return elevio.HallDown
	}
	return elevio.HallUp
}

type OrderHandler struct {
	wvChan    chan statesync.Worldview
	trigger   chan struct{}
	actionCah chan any
}

func NewOrderHandler(wvChan chan statesync.Worldview, trigger chan struct{}, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		wvChan:    wvChan,
		trigger:   trigger,
		actionCah: actionChan,
	}
}

func (o *OrderHandler) Run() {
	// prevHC := make(map[int][2]statesync.HallCallPairState)

	// for wv := range o.wvChan {
	// 	hallCalls := wv.GetAllHallCalls()

	// 	// Propagate light changes driven by any hall call state transition
	// 	for floor := range hallCalls {
	// 		for d := range hallCalls[floor] {
	// 			dir := statesync.HallCallDir(d)
	// 			prevState := prevHC[floor][dir]
	// 			currentState := hallCalls[floor][dir]

	// 			if prevState.State != currentState.State {
	// 				btn := HallDirToButtonType(dir)
	// 				switch currentState.State {
	// 				case statesync.HSNone:
	// 					o.actionCah <- elevator.SingleLightAction{ButtonType: btn, Floor: floor, State: elevator.LSOff}
	// 				case statesync.HSAvailable, statesync.HSProcessing:
	// 					o.actionCah <- elevator.SingleLightAction{ButtonType: btn, Floor: floor, State: elevator.LSOn}
	// 				}
	// 			}
	// 			temp := prevHC[floor]
	// 			temp[dir] = currentState
	// 			prevHC[floor] = temp
	// 		}
	// 	}

	for wv := range o.wvChan {
		hallCalls := wv.GetAllHallCalls()

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				if hallCall[dir].State == statesync.HSNone {
					continue
				}

				isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= len(wv.ElevatorStates)
				isAvailable := hallCall[dir].State == statesync.HSAvailable

				if hallCall[dir].State == statesync.HSProcessing && hallCall[dir].By == wv.LocalID {
					o.trigger <- struct{}{}
				}

				if !isConfirmedByAll || !isAvailable {
					// slog.Info("[RunCost]", "confirmed by all", isConfirmedByAll, "available", isAvailable)
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
				slog.Warn("[RunCost] Order picked up", "by", winner.id)
				o.actionCah <- elevator.SingleLightAction{ButtonType: HallDirToButtonType(statesync.HallCallDir(dir)), Floor: floor, State: elevator.LSOn}
			}
		}
	}
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
		}
	}
	return winner
}
