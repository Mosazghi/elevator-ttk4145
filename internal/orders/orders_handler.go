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
				isConfirmed := hallCall[dir].State == statesync.HallCallStateConfirmed

				if !isConfirmed {
					continue
				}
				slog.Debug("Calcing cost function", "floor", floor, "dir", dir)

				
				winner := CalculateCost(o.wv, floor, statesync.HallCallDir(dir))
				if winner.id == o.wv.LocalID {
					err := o.wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[worldview error", "err", err)
					}
					slog.Info("won, set to processing", "floor", floor, "dir", dir, "id", winner.id)

					o.trigger <- controller.CTSOrderUpdate
				}
				slog.Info("order picked up", "floor", floor, "dir", dir, "by", winner.id)
			}
		}
	}
}
