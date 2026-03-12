package controller

import (
	"log/slog"
	"math"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
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
	doorDuration  time.Duration
}

func NewController(wv *statesync.Worldview, actionChan chan any, ctrlTrigger chan ControllerTriggerSrc, hcLightChan chan statesync.Order) *Controller {
	return &Controller{
		wv:            wv,
		actionChan:    actionChan,
		triggerChan:   ctrlTrigger,
		hcLightChan:   hcLightChan,
		doorTimerChan: nil,
		doorDuration:  config.DoorOpenTime,
	}
}

func (ctrl *Controller) OnFloorArrival() {
	elev := ctrl.wv.GetRemoteElevator()
	ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorDoorOpen, Direction: elevio.MDStop}
	ctrl.actionChan <- elevator.DoorAction{Open: true}

	ctrl.clearAllOrdersAtFloor(elev.CurrentFloor)
	ctrl.doorTimerChan = time.After(ctrl.doorDuration)
}

// clearAllOrdersAtFloor completes all cab calls and hall calls at the given floor
func (ctrl *Controller) clearAllOrdersAtFloor(floor int) {
	elev := ctrl.wv.GetRemoteElevator()

	if elev.CabCalls[floor] {
		ctrl.wv.SetCabCall(floor, false)
		ctrl.actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: floor, State: false}
	}

	// FIXME: Find a better solution than blocking for 500ms!
	time.Sleep(150 * time.Millisecond) // Ensure other nodes have time to process the hall call before completing it
	hcs := ctrl.wv.GetAllHallCalls()
	for dir := range hcs[floor] {
		isOurs := hcs[floor][dir].By == elev.ID && hcs[floor][dir].State == statesync.HSProcessing
		if isOurs {
			err := ctrl.wv.CompleteHallCall(floor, statesync.HallCallDir(dir))
			if err != nil {
				slog.Error("Failed to complete hall call", "error", err)
			}
		}
	}
}

func (ctrl *Controller) Start() {
	for {
		select {
		case order := <-ctrl.hcLightChan:
			ctrl.actionChan <- elevator.LightAction{ButtonType: shared.HallDirToButtonType(order.Dir), Floor: order.Floor, State: !order.Completed}
		case <-ctrl.doorTimerChan:
			elev := ctrl.wv.GetRemoteElevator()

			if elev.IsObstructed {
				ctrl.doorTimerChan = time.After(ctrl.doorDuration)
			} else {
				ctrl.doorTimerChan = nil
				ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorIdle, Direction: elevio.MDStop}
				ctrl.actionChan <- elevator.DoorAction{Open: false}
			}
		case triggerSrc := <-ctrl.triggerChan:
			elev := ctrl.wv.GetRemoteElevator()
			closestOrder := FetchClosestOrder(ctrl.wv)

			switch elev.Behavior {
			case elevator.BehaviorDoorOpen:
				if closestOrder.AtFloor(elev.CurrentFloor) {
					ctrl.clearAllOrdersAtFloor(elev.CurrentFloor)
				}
			case elevator.BehaviorMoving:
				if closestOrder.Empty() {
					ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorIdle, Direction: elevio.MDStop}
					continue
				}

				if closestOrder.AtFloor(elev.CurrentFloor) && triggerSrc == CTSFArrivalFloor {
					ctrl.OnFloorArrival()
				}
			case elevator.BehaviorIdle:
				if closestOrder.Empty() || (elev.IsObstructed && elev.Behavior == elevator.BehaviorDoorOpen) {
					continue
				}

				if closestOrder.AtFloor(elev.CurrentFloor) {
					ctrl.OnFloorArrival()
					continue
				}

				ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BehaviorMoving, Direction: closestOrder.MotorDirection}
			}
		}
	}
}

func FetchClosestOrder(worldView *statesync.Worldview) CurrentOrder {
	closestCabCall := FindClosestCabCall(worldView)
	closestHallCall := FindClosestHallCall(worldView)
	localElevator := worldView.GetRemoteElevator()

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
