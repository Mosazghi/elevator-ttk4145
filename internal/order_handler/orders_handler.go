// Package order_handler implements the logic for claiming confirmed hall calls and notifying the controller.
package order_handler

import (
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// OrderHandler periodically evaluates confirmed hall calls and claims the ones
// assigned to the local elevator.
type OrderHandler struct {
	worldview         *statesync.Worldview
	controllerTrigger chan controller.ControllerTriggerSrc
	actionChan        chan any
}

// NewOrderHandler constructs the order handler loop dependencies.
func NewOrderHandler(wv *statesync.Worldview, trigger chan controller.ControllerTriggerSrc, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		worldview:         wv,
		controllerTrigger: trigger,
		actionChan:        actionChan,
	}
}

// Run polls hall calls and transitions locally won confirmed calls to
// processing, then notifies the controller.
func (oh *OrderHandler) Run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		hallCalls := oh.worldview.GetAllHallCalls()

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				isConfirmed := hallCall[dir].State == statesync.HallCallStateConfirmed

				if !isConfirmed {
					continue
				}

				winnerID, err := CalculateCost(oh.worldview, floor, statesync.HallCallDir(dir))

				if err != nil {
					slog.Error("failed to calculate cost for hall call", "floor", floor, "dir", dir, "error", err)
					continue
				}

				if winnerID == oh.worldview.LocalID {
					err := oh.worldview.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("failed to process hall call", "err", err)
					}
					slog.Info("won, set to processing", "floor", floor, "dir", dir, "id", winnerID)

					oh.controllerTrigger <- controller.CTSOrderUpdate
				}
				slog.Info("order picked up", "floor", floor, "dir", dir, "by", winnerID)
			}
		}
	}
}
