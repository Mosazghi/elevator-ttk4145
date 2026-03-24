// Package controller makes decisions for the local elevator state machine.
//
// Controller observes world state via statesync.Worldview and responds to triggers
// from orchestrator by emitting elevator actions (move/stop/light/door) on an action
// channel. It also handles order completion, light updates, and the door timer.
//
// The controller works inside a single goroutine through StartHandlingRequests.
package controller

import (
	"log/slog"
	"math"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
)

// A Controller dictate which elevator action is needed to be performed for serving an order.
//
// Actions are strictly passed though actionChan, while the other channels (execpt doorTimerChan) are used
// to dictate when to use the Controller
type Controller struct {
	worldview       *statesync.Worldview
	actionChan      chan any
	triggerChan     chan ControllerTriggerSrc
	orderUpdateChan chan statesync.Order
	doorTimerChan   <-chan time.Time
	doorDuration    time.Duration
}

// NewController creates a new instance of a Controller
func NewController(worldView *statesync.Worldview, actionChan chan any, ctrlTrigger chan ControllerTriggerSrc, orderUpdateChan chan statesync.Order) *Controller {
	return &Controller{
		worldview:       worldView,
		actionChan:      actionChan,
		triggerChan:     ctrlTrigger,
		orderUpdateChan: orderUpdateChan,
		doorTimerChan:   nil,
		doorDuration:    config.DoorOpenTime,
	}
}

// OnFloorArrival starts the arrival sequence when completing an order
func (ctrl *Controller) OnFloorArrival(order CurrentOrder) error {
	if order.Empty() {
		return nil
	}
	ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BehaviorDoorOpen}
	ctrl.actionChan <- elevator.DoorAction{Open: true}

	return ctrl.clearOrderAtFloor(order)
}

// clearOrderAtFloor completes the currently handled order
func (ctrl *Controller) clearOrderAtFloor(order CurrentOrder) error {
	err := order.Complete(ctrl.worldview)
	if err != nil {
		return err
	}

	ctrl.doorTimerChan = time.After(ctrl.doorDuration)
	return nil
}

// StartHandlingRequests provides actions when requested though the following channels:
// orderUpdateChan, doorDuration, triggerChan
//
// This function is designed as a goroutine which handles requests.
// These requests typically occur when arriving at a new floor or creating a new order.
func (ctrl *Controller) StartHandlingRequests() {
	for {
		select {
		case order := <-ctrl.orderUpdateChan:
			ctrl.actionChan <- elevator.LightAction{ButtonType: order.Type, Floor: order.Floor, State: !order.Completed}
		case <-ctrl.doorTimerChan:
			elev := ctrl.worldview.GetRemoteElevatorState()

			if elev.IsObstructed {
				ctrl.doorTimerChan = time.After(ctrl.doorDuration)
			} else {
				ctrl.doorTimerChan = nil
				ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BehaviorIdle}
				ctrl.actionChan <- elevator.DoorAction{Open: false}
			}
		case triggerSrc := <-ctrl.triggerChan:
			elev := ctrl.worldview.GetRemoteElevatorState()
			closestOrder := FetchClosestOrder(ctrl.worldview)

			switch elev.Behavior {

			case elevator.BehaviorDoorOpen:
				if closestOrder.AtFloor(elev.CurrentFloor) {
					err := ctrl.clearOrderAtFloor(closestOrder)
					if err != nil {
						slog.Error("error clearing at floor", "err", err)
					}
				}
			case elevator.BehaviorMoving:
				if closestOrder.Empty() || closestOrder.OppositeDirection(elev.Direction) {
					ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BehaviorIdle}
					continue
				}

				if triggerSrc == ControllerTriggerSrcOrderUpdate {
					ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorMoving, Direction: closestOrder.MotorDirection}
				}

				if closestOrder.AtFloor(elev.CurrentFloor) && triggerSrc == ControllerTriggerSrcArrivalFloor {
					ctrl.OnFloorArrival(closestOrder)
				}
			case elevator.BehaviorIdle:
				if closestOrder.Empty() {
					continue
				}

				if closestOrder.AtFloor(elev.CurrentFloor) {
					ctrl.OnFloorArrival(closestOrder)
					continue
				}

				ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorMoving, Direction: closestOrder.MotorDirection}
			}
		}
	}
}

// FetchClosestOrder Finds the closest order to the local elevator.
//
// The function finds the closest cab call and hall call and decides
// which is the closest order.
func FetchClosestOrder(worldView *statesync.Worldview) CurrentOrder {
	closestCabCall, cabCallCost := FindClosestCabCall(worldView)
	closestHallCall, hallCallCost := FindClosestHallCall(worldView)

	if !closestCabCall.Empty() && closestHallCall.Empty() {
		return closestCabCall
	}

	if closestCabCall.Empty() && !closestHallCall.Empty() {
		return closestHallCall
	}

	if cabCallCost < hallCallCost {
		return closestCabCall
	} else {
		return closestHallCall
	}
}

// FindClosestCabCall Searches for the closest cab call to the local elevator.
//
// This function calculates the cost of arriving at said cab call, by factoring in
// the direction and distance of the order.
func FindClosestCabCall(worldview *statesync.Worldview) (CurrentOrder, int) {
	var motorDirection elevio.MotorDirection
	localElevator := worldview.GetRemoteElevatorState()
	closestOrder := NewOrder()
	lowestCost := math.MaxInt

	for floor, found := range localElevator.CabCalls {
		cost := 0
		if !found {
			continue
		}

		if localElevator.CurrentFloor < floor {
			motorDirection = elevio.MotorDirectionUp
		} else {
			motorDirection = elevio.MotorDirectionDown
		}

		if localElevator.CurrentFloor == floor {
			motorDirection = localElevator.Direction
		}

		if localElevator.IsOppositeMotorDirection(motorDirection) {
			cost += PenaltyOppositeMotorDirection
		}

		cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

		if cost < lowestCost {
			lowestCost = cost
			closestOrder.Update(floor, elevio.Cab, motorDirection, statesync.HallCallDirectionDown) // choose arbritaty hall call direction
		}
	}
	return closestOrder, lowestCost
}

// FindClosestHallCall Searches for the closest hall call to the local elevator.
//
// This function uses the distance, relative direction and hall call direction
// to determine the closest hall call.
func FindClosestHallCall(worldview *statesync.Worldview) (CurrentOrder, int) {
	var motorDirection elevio.MotorDirection
	var orderType elevio.ButtonType
	var hallCallDirection statesync.HallCallDirection
	localElevator := worldview.GetRemoteElevatorState()
	closestOrder := NewOrder()
	lowestCost := math.MaxInt

	for floor, hallCall := range worldview.GetAllHallCalls() {
		for direction := range hallCall {
			cost := 0

			if hallCall[direction].AssignedBy != localElevator.ID {
				continue
			}

			if localElevator.CurrentFloor < floor {
				motorDirection = elevio.MotorDirectionUp
			} else {
				motorDirection = elevio.MotorDirectionDown
			}

			if localElevator.CurrentFloor == floor {
				motorDirection = localElevator.Direction
			}

			if localElevator.IsOppositeMotorDirection(motorDirection) {
				cost += PenaltyOppositeMotorDirection
			}

			if localElevator.IsOppositeHallCallDirection(statesync.HallCallDirection(direction)) {
				cost += PenaltyOppositeHallCallDirection
			}

			hallCallDirection = statesync.HallCallDirection(direction)

			orderType = statesync.HallDirToButtonType(hallCallDirection)

			cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

			if cost < lowestCost {
				lowestCost = cost
				closestOrder.Update(floor, orderType, motorDirection, hallCallDirection)
			}
		}
	}

	return closestOrder, lowestCost
}
