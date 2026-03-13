package orders

import (
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type ElevatorCost struct {
	id    int
	floor int
	cost  int
}

type OrderHandler struct {
	wv        *statesync.Worldview
	trigger   chan controller.ControllerTriggerSrc
	actionCah chan any
}

func NewOrderHandler(wv *statesync.Worldview, trigger chan controller.ControllerTriggerSrc, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		wv:        wv,
		trigger:   trigger,
		actionCah: actionChan,
	}
}

func (o *OrderHandler) Run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		hallCalls := o.wv.GetAllHallCalls()

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				isAvailable := hallCall[dir].State == statesync.HSAvailable

				if !isAvailable {
					continue
				}

				aliveCount := 0
				for _, elev := range o.wv.ElevatorStates {
					if elev.Alive {
						aliveCount++
					}
				}

				isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= aliveCount

				if !isConfirmedByAll {
					continue
				}

				winner := CalculateCost(o.wv, floor, statesync.HallCallDir(dir))
				if winner.id == o.wv.LocalID {
					err := o.wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
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
