package orders

import (
	"log/slog"
	"math"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type ElevatorCost struct {
	ID    int
	Floor int
	Cost  int
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

func (o *OrderHandler) GetWvChan() chan statesync.Worldview {
	return o.wvChan
}

func (o *OrderHandler) GetTrigger() chan struct{} {
	return o.trigger
}

func (o *OrderHandler) GetActionChan() chan any {
	return o.actionCah
}

func (o *OrderHandler) Run() {
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
					select {
					case o.trigger <- struct{}{}:
						slog.Info("Triggered from OrderHandler")
					default:
					}
				}

				if !isConfirmedByAll || !isAvailable {
					// slog.Info("[RunCost]", "confirmed by all", isConfirmedByAll, "available", isAvailable)
					continue
				}

				// slog.Info("[RunCost] Calculating cost")
				winner := CalculateCost(&wv, floor, statesync.HallCallDir(dir))
				if winner.ID == wv.LocalID {
					err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[RunCost] Got worldview error", "error", err)
					}
					slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir, "id", winner.ID)
				}
				slog.Warn("[RunCost] Order picked up", "by", winner.ID)
				o.actionCah <- elevator.SingleLightAction{ButtonType: HallDirToButtonType(statesync.HallCallDir(dir)), Floor: floor, State: true}
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
		currentElevatorCost.ID = id

		if isObstructed {
			// slog.Info("[CalculateCost] Elevator is obstructed")
			currentElevatorCost.Cost += PenaltyObstructed
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			// slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.Cost += PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			// slog.Info("[CalculateCost] Elevator moves oposite direction", "dir", dir)
			currentElevatorCost.Cost += PenaltyWrongDirection
		}

		distance := int(math.Abs(float64(elev.CurrentFloor - floor)))

		currentElevatorCost.Cost += distance

		currentElevatorCost.Cost += id

		// slog.Info("[CalculateCost] found valid elevator", "cost", cost, "floor", floor, "id", currentElevatorCost.ID)

		if currentElevatorCost.Cost < winner.Cost {
			winner.ID = currentElevatorCost.ID
			winner.Cost = currentElevatorCost.Cost
			winner.Floor = currentElevatorCost.Floor
		}
	}
	return winner
}
