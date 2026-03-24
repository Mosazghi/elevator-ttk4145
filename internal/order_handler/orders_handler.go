// Package order_handler implements the logic for claiming confirmed hall calls and notifying the controller.
package order_handler

import (
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// OrderHandler periodically (every 100ms) evaluates confirmed hall calls and claims the ones
// assigned to the local elevator.
type OrderHandler struct {
	worldview         *statesync.Worldview
	controllerTrigger chan controller.ControllerTriggerSrc
	actionChan        chan any
}

// NewOrderHandler constructs the order handler loop dependencies.
func NewOrderHandler(worldView *statesync.Worldview, trigger chan controller.ControllerTriggerSrc, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		worldview:         worldView,
		controllerTrigger: trigger,
		actionChan:        actionChan,
	}
}

// StartServing polls hall calls and transitions locally won confirmed calls to
// processing, then notifies the controller.
func (orderHandler *OrderHandler) StartServing() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		hallCalls := orderHandler.worldview.GetAllHallCalls()

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				isConfirmed := hallCall[dir].State == statesync.HallCallStateConfirmed

				if !isConfirmed {
					continue
				}

				winnerID, err := CalculateCost(orderHandler.worldview, floor, statesync.HallCallDirection(dir))

				if err != nil {
					slog.Error("failed to calculate cost for hall call", "floor", floor, "dir", dir, "error", err)
					continue
				}

				if winnerID == orderHandler.worldview.LocalID {
					err := orderHandler.worldview.ProcessHallCall(floor, statesync.HallCallDirection(dir))
					if err != nil {
						slog.Error("failed to process hall call", "err", err)
					}
					slog.Info("won, set to processing", "floor", floor, "dir", dir, "id", winnerID)

					orderHandler.controllerTrigger <- controller.ControllerTriggerSrcOrderUpdate
				}
				slog.Info("order picked up", "floor", floor, "dir", dir, "by", winnerID)
			}
		}
	}
}
