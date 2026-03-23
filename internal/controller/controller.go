package controller

import (
	"math"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/Mosazghi/elevator-ttk4145/shared"
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

func (ctrl *Controller) OnFloorArrival(order CurrentOrder) {
	if order.Empty() {
		return
	}
	ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BDoorOpen}
	ctrl.actionChan <- elevator.DoorAction{Open: true}

	ctrl.clearAllOrdersAtFloor(order)
	ctrl.doorTimerChan = time.After(ctrl.doorDuration)
}

// clearAllOrdersAtFloor completes all cab calls and hall calls at the given floor
func (ctrl *Controller) clearAllOrdersAtFloor(order CurrentOrder) {
	elev := ctrl.wv.GetRemoteElevator()
	floor := order.Floor

	if elev.CabCalls[floor] {
		// NOTE: Have to set cab call to false here as well if the elevator is on the same floor,
		// and just reconnected
		ctrl.wv.SetCabCall(floor, false)
		ctrl.actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: floor, State: false}
	}

	time.Sleep(500 * time.Millisecond) // Ensure other nodes have time to process the hall call before completing it
	order.Complete(ctrl.wv)
}

func (ctrl *Controller) Start() {
	for {
		select {
		case order := <-ctrl.hcLightChan:
			ctrl.actionChan <- elevator.LightAction{ButtonType: HallDirToButtonType(order.Direction), Floor: order.Floor, State: !order.Completed}
		case <-ctrl.doorTimerChan:
			elev := ctrl.wv.GetRemoteElevator()

			if elev.IsObstructed {
				ctrl.doorTimerChan = time.After(ctrl.doorDuration)
			} else {
				ctrl.doorTimerChan = nil
				ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BIdle}
				ctrl.actionChan <- elevator.DoorAction{Open: false}
			}
		case triggerSrc := <-ctrl.triggerChan:
			elev := ctrl.wv.GetRemoteElevator()
			closestOrder := FetchClosestOrder(ctrl.wv)

			switch elev.Behavior {

			case elevator.BDoorOpen:
				if closestOrder.AtFloor(elev.CurrentFloor) {
					ctrl.clearAllOrdersAtFloor(closestOrder)
				}
			case elevator.BMoving:
				if closestOrder.Empty() || closestOrder.OppositeDirection(elev.Direction) {
					ctrl.actionChan <- elevator.StopAction{Behavior: elevator.BIdle}
					continue
				}

				if closestOrder.AtFloor(elev.CurrentFloor) && triggerSrc == CTSFArrivalFloor {
					ctrl.OnFloorArrival(closestOrder)
				}
			case elevator.BIdle:
				if closestOrder.Empty() {
					continue
				}

				if closestOrder.AtFloor(elev.CurrentFloor) {
					ctrl.OnFloorArrival(closestOrder)
					continue
				}

				ctrl.actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: closestOrder.MotorDirection}
			}
		}
	}
}

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

func FindClosestCabCall(wv *statesync.Worldview) (CurrentOrder, int) {
	var motorDirection elevio.MotorDirection
	localElevator := wv.GetRemoteElevator()
	closestOrder := NewOrder()
	bestCost := math.MaxInt

	for floor, found := range localElevator.CabCalls {
		cost := 0
		if !found {
			continue
		}

		if localElevator.CurrentFloor < floor {
			motorDirection = elevio.MDUp
		} else {
			motorDirection = elevio.MDDown
		}

		if localElevator.CurrentFloor == floor {
			motorDirection = localElevator.Direction
		}

		// NOTE: ask about this
		// Kinda the same as closestOrder.WrongDirection ?? But imo more clear
		if localElevator.IsOppositeMotorDirection(motorDirection) {
			cost += shared.PenaltyOppositeMotorDirection
		}

		cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

		if cost < bestCost {
			bestCost = cost
			closestOrder.Update(floor, elevio.Cab, motorDirection, statesync.HDDown) // TODO: use none direction to be clear
		}
	}
	return closestOrder, bestCost
}

func FindClosestHallCall(wv *statesync.Worldview) (CurrentOrder, int) {
	var motorDirection elevio.MotorDirection
	var orderType elevio.ButtonType
	var hallCallDirection statesync.HallCallDir
	localElevator := wv.GetRemoteElevator()
	closestOrder := NewOrder()
	bestCost := math.MaxInt

	for floor, hallCall := range wv.GetAllHallCalls() {
		for direction := range hallCall {
			cost := 0

			if hallCall[direction].AssignedBy != localElevator.ID {
				continue
			}

			if localElevator.CurrentFloor < floor {
				motorDirection = elevio.MDUp
			} else {
				motorDirection = elevio.MDDown
			}

			if localElevator.CurrentFloor == floor {
				motorDirection = localElevator.Direction
			}

			if localElevator.IsOppositeMotorDirection(motorDirection) {
				cost += shared.PenaltyOppositeMotorDirection
			}

			if localElevator.IsOppositeHallCallDirection(statesync.HallCallDir(direction)) {
				cost += shared.PenaltyOppositeHallCallDirection
			}

			hallCallDirection = statesync.HallCallDir(direction)
			orderType = HallDirToButtonType(hallCallDirection)

			cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

			if cost < bestCost {
				bestCost = cost
				closestOrder.Update(floor, orderType, motorDirection, hallCallDirection)
			}
		}
	}

	return closestOrder, bestCost
}
