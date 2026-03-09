package orders

import (
	"log/slog"
	"math"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/Mosazghi/elevator-ttk4145/shared"
)

type ElevatorCost struct {
	id    int
	floor int
	cost  int
}

type OrderHandler struct {
	wvChan    chan statesync.Worldview
	trigger   chan controller.ControllerTriggerSrc
	actionCah chan any
}

func NewOrderHandler(wvChan chan statesync.Worldview, trigger chan controller.ControllerTriggerSrc, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		wvChan:    wvChan,
		trigger:   trigger,
		actionCah: actionChan,
	}
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

				if !isConfirmedByAll || !isAvailable {
					continue
				}

				winner := CalculateCost(&wv, floor, statesync.HallCallDir(dir))
				if winner.id == wv.LocalID {
					err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[RunCost] Got worldview error", "error", err)
					}
					slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir, "id", winner.id)

					o.trigger <- controller.CTSOrderUpdate
				}
				slog.Warn("[RunCost] Order picked up", "by", winner.id)
			}
		}
	}
}

func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}

	for id, elev := range wv.ElevatorStates {
		currentElevatorCost := ElevatorCost{-1, wv.NumFloors + 1, 0}
		isObstructed := elev.IsObstructed

		currentElevatorCost.id = id

		if isObstructed {
			currentElevatorCost.cost += shared.PenaltyObstructed
		}

		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		distance := int(math.Abs(float64(elev.CurrentFloor - floor)))

		currentElevatorCost.cost += distance

		slog.Error("[CalculateCost] Cost for elevator", "id", id, "cost", currentElevatorCost.cost, "distance", distance, "isObstructed", isObstructed, "currentDir", elev.Direction)

		if currentElevatorCost.cost < winner.cost || (currentElevatorCost.cost == winner.cost && currentElevatorCost.id < winner.id) {
			winner.id = currentElevatorCost.id
			winner.cost = currentElevatorCost.cost
			winner.floor = currentElevatorCost.floor
		}
	}
	return winner
}
