package orders

import (
	"log/slog"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
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
				isNone := hallCall[dir].State == statesync.HSNone
				isAvailable := hallCall[dir].State == statesync.HSAvailable

				if isNone || !isAvailable {
					continue
				}

				aliveCount := 0
				for _, elev := range wv.ElevatorStates {
					if elev.Alive {
						aliveCount++
					}
				}

				isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= aliveCount

				if !isConfirmedByAll {
					continue
				}

				winner := CalculateCost(&wv, floor, statesync.HallCallDir(dir))
				if winner.id == wv.LocalID {
					err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[worldview error", "err", err)
					}
					slog.Info("set to processing", "floor", floor, "Direction", dir, "id", winner.id)

					o.trigger <- controller.CTSOrderUpdate
				}
				slog.Info("order picked up", "floor", floor, "dir", dir, "by", winner.id)
			}
		}
	}
}
