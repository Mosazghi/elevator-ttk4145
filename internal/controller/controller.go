package controller

import (
	"log/slog"
	"math"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/Mosazghi/elevator-ttk4145/shared"
)

type ControllerTriggerSrc int

const (
	CTSFArrivalFloor ControllerTriggerSrc = iota
	CTSOrderUpdate
	CTSHCLights
)

type Controller struct {
	wv            *statesync.Worldview
	actionChan    chan any
	triggerChan   chan ControllerTriggerSrc
	hcLightChan   chan statesync.Order
	doorTimerChan <-chan time.Time
}

func NewController(wv *statesync.Worldview, actionChan chan any, ctrlTrigger chan ControllerTriggerSrc, hcLightChan chan statesync.Order) *Controller {
	return &Controller{
		wv:            wv,
		actionChan:    actionChan,
		triggerChan:   ctrlTrigger,
		hcLightChan:   hcLightChan,
		doorTimerChan: nil,
	}
}

func (ctrl *Controller) OnFloorArrival() {
	remote := ctrl.wv.GetRemoteElevator()
	slog.Debug("Should stop")
	ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BDoorOpen, Direction: elevio.MDStop}
	ctrl.actionChan <- elevator.DoorAction{Open: true}
	ctrl.actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: remote.CurrentFloor, State: false}

	ctrl.doorTimerChan = time.After(3 * time.Second)
}

func (ctrl *Controller) Start() {
	for {
		select {
		case order := <-ctrl.hcLightChan:
			ctrl.actionChan <- elevator.LightAction{ButtonType: shared.HallDirToButtonType(order.Dir), Floor: order.Floor, State: !order.Completed}
		case <-ctrl.doorTimerChan:
			ctrl.doorTimerChan = nil
			ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
			ctrl.actionChan <- elevator.DoorAction{Open: false}
		case triggerSrc := <-ctrl.triggerChan:
			switch triggerSrc {
			case CTSOrderUpdate:
				localElevator := ctrl.wv.GetRemoteElevator()
				closestOrder := FetchClosestOrder(ctrl.wv)
				if closestOrder.Empty() {
					continue
				}

				if localElevator.AllowedToServe() && closestOrder.AtFloor(localElevator.CurrentFloor) {
					ctrl.OnFloorArrival()
					closestOrder.Complete(ctrl.wv)
					currFloor := localElevator.CurrentFloor
					ctrl.wv.SetCabCall(currFloor, false)
					hcs := ctrl.wv.GetAllHallCalls()
					for dir := range hcs[currFloor] {
						isOur := hcs[currFloor][dir].By == localElevator.ID && hcs[currFloor][dir].State == statesync.HSProcessing
						slog.Info("[Completing hallcall]", "floor", currFloor, "direction", dir, "isOur", isOur)
						if isOur {
							ctrl.wv.CompleteHallCall(currFloor, statesync.HallCallDir(dir))
						}
					}
					continue
				}

				if localElevator.AllowedToServe() {
					ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: closestOrder.MotorDirection}
				}
			case CTSFArrivalFloor:
				localElevator := ctrl.wv.GetRemoteElevator()
				closestOrder := FetchClosestOrder(ctrl.wv)
				// slog.Info("[FetchClosestOrder] got closestOrder", "floor", closestOrder.Floor)

				if closestOrder.Empty() {
					// slog.Debug("[Controller] No calls available, stopping")
					ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
					continue
				}

				if closestOrder.AtFloor(localElevator.CurrentFloor) {
					// slog.Info("[atFloor] true", "floor", closestOrder.Floor, "type", closestOrder.Type, "direction", closestOrder.MotorDirection)
					ctrl.OnFloorArrival()
					closestOrder.Complete(ctrl.wv)
				}
			}
		}
	}
}

func FetchClosestOrder(worldView *statesync.Worldview) CurrentOrder {
	closestCabCall := FindClosestCabCall(worldView)
	closestHallCall := FindClosestHallCall(worldView)
	localElevator := worldView.GetRemoteElevator()

	// slog.Info("[closestHallCall]", "floor", closestHallCall.Floor)
	// slog.Info("[closestCabCall]", "floor", closestCabCall.Floor)

	if !closestCabCall.Empty() && closestHallCall.Empty() {
		return closestCabCall
	}

	if closestCabCall.Empty() && !closestHallCall.Empty() {
		return closestHallCall
	}

	cabCallDistance := int(math.Abs(float64(closestCabCall.Floor - localElevator.CurrentFloor)))
	hallCallDistance := int(math.Abs(float64(closestHallCall.Floor - localElevator.CurrentFloor)))

	if cabCallDistance < hallCallDistance {
		return closestCabCall
	} else {
		return closestHallCall
	}
}

func FindClosestCabCall(wv *statesync.Worldview) CurrentOrder {
	var motorDirection elevio.MotorDirection
	localElevator := wv.GetRemoteElevator()
	closestOrder := NewOrder()
	bestCost := math.MaxInt

	for floor, found := range localElevator.CabCalls {
		cost := 0
		if !found {
			continue
		}

		if localElevator.WrongDirection(floor) {
			cost += shared.PenaltyWrongDirection
		}

		if localElevator.CurrentFloor < floor {
			motorDirection = elevio.MDUp
		} else {
			motorDirection = elevio.MDDown
		}

		cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

		if cost < bestCost {
			bestCost = cost
			closestOrder.Update(floor, elevio.Cab, motorDirection, statesync.HDDown) // TODO: use none direction to be clear
		}
	}
	return closestOrder
}

func FindClosestHallCall(wv *statesync.Worldview) CurrentOrder {
	var motorDirection elevio.MotorDirection
	var orderType elevio.ButtonType
	var hallCallDirection statesync.HallCallDir
	localElevator := wv.GetRemoteElevator()
	closestOrder := NewOrder()
	bestCost := math.MaxInt

	for floor, hallCall := range wv.GetAllHallCalls() {
		for direction := range hallCall {
			cost := 0

			if hallCall[direction].By != localElevator.ID {
				continue
			}

			if localElevator.WrongDirection(floor) {
				cost += shared.PenaltyWrongDirection
			}

			if localElevator.CurrentFloor < floor {
				motorDirection = elevio.MDUp
			} else {
				motorDirection = elevio.MDDown
			}

			hallCallDirection = statesync.HallCallDir(direction)
			orderType = shared.HallDirToButtonType(hallCallDirection)

			cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

			if cost < bestCost {
				bestCost = cost
				closestOrder.Update(floor, orderType, motorDirection, hallCallDirection)
			}
		}
	}

	return closestOrder
}
