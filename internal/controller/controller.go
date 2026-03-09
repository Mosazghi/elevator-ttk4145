package controller

import (
	"log/slog"
	"math"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/orders"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type Order struct {
	Floor             int
	Type              elevio.ButtonType
	TravelDirection   elevio.MotorDirection
	HallCallDirection statesync.HallCallDir
}

func (order *Order) Complete(worldView *statesync.Worldview) {
	if order.Type == elevio.Cab {
		worldView.SetCabCall(order.Floor, false)
	} else {
		err := worldView.CompleteHallCall(order.Floor, order.HallCallDirection)
		if err != nil {
			slog.Error("[CompleteHallCall] in order.Complete", "error", err)
		}
		slog.Warn("[Completing hallcall]", "floor", order.Floor, "direction", order.HallCallDirection)
	}
}

func (order *Order) Empty() bool {
	return order.Floor == -1
}

func (order *Order) AtFloor(floor int) bool {
	if floor == -1 {
		return false
	}
	return order.Floor == floor
}

func (order *Order) Update(floor int, orderType elevio.ButtonType, Motordirection elevio.MotorDirection, hallCallDirection statesync.HallCallDir) {
	order.Floor = floor
	order.Type = orderType
	order.TravelDirection = Motordirection
	order.HallCallDirection = hallCallDirection
}

func NewOrder() Order {
	return Order{
		Floor:             -1,
		Type:              elevio.Cab,
		TravelDirection:   elevio.MDStop,
		HallCallDirection: statesync.HDDown,
	}
}

func ArrivalSequence(wv *statesync.Worldview, actionChan chan any) {
	remote := wv.GetRemoteElevator()
	slog.Debug("Should stop")
	actionChan <- elevator.MoveAction{Behavior: elevator.BDoorOpen, Direction: elevio.MDStop}
	actionChan <- elevator.DoorAction{Open: true}
	actionChan <- elevator.LightAction{ButtonType: elevio.Cab, Floor: remote.CurrentFloor, State: false}

	time.AfterFunc(3*time.Second, func() {
		actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
		actionChan <- elevator.DoorAction{Open: false}
		slog.Debug("Finished stopping")
	})
}

func Start(wv *statesync.Worldview, arriveAtFloor chan struct{}, actionChan chan any, newOrder chan struct{}, hcLightChan chan statesync.Order) {
	for {
		select {
		case order := <-hcLightChan:
			actionChan <- elevator.LightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: !order.Completed}
		case <-newOrder:
			localElevator := wv.GetRemoteElevator()
			closestOrder := FetchClosestOrder(wv)

			if closestOrder.Empty() {
				continue
			}

			if localElevator.AllowedToServe() && closestOrder.AtFloor(localElevator.CurrentFloor) {
				ArrivalSequence(wv, actionChan)
				time.Sleep(500 * time.Millisecond)
				// time.AfterFunc(500*time.Millisecond, func() {
				closestOrder.Complete(wv)
				// })
				continue
			}

			if localElevator.AllowedToServe() {
				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: closestOrder.TravelDirection}
			}

		case <-arriveAtFloor:
			localElevator := wv.GetRemoteElevator()
			closestOrder := FetchClosestOrder(wv)
			slog.Info("[FetchClosestOrder] got closestOrder", "floor", closestOrder.Floor)

			if closestOrder.Empty() {
				slog.Debug("[Controller] No calls available, stopping")
				actionChan <- elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop}
				continue
			}

			if closestOrder.AtFloor(localElevator.CurrentFloor) {
				slog.Info("[atFloor] true", "floor", closestOrder.Floor, "type", closestOrder.Type, "direction", closestOrder.TravelDirection)
				ArrivalSequence(wv, actionChan)
				time.Sleep(500 * time.Millisecond)
				closestOrder.Complete(wv)
			}
		}
	}
}

func FetchClosestOrder(worldView *statesync.Worldview) Order {
	closestCabCall := FindClosestCabCall(worldView)
	closestHallCall := FindClosestHallCall(worldView)
	localElevator := worldView.GetRemoteElevator()

	slog.Info("[closestHallCall]", "floor", closestHallCall.Floor)
	slog.Info("[closestCabCall]", "floor", closestCabCall.Floor)

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

func FindClosestCabCall(wv *statesync.Worldview) Order {
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
			cost += orders.PenaltyWrongDirection
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

func FindClosestHallCall(wv *statesync.Worldview) Order {
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
				cost += orders.PenaltyWrongDirection
			}

			if localElevator.CurrentFloor < floor {
				motorDirection = elevio.MDUp
			} else {
				motorDirection = elevio.MDDown
			}

			hallCallDirection = statesync.HallCallDir(direction)
			orderType = orders.HallDirToButtonType(hallCallDirection)

			cost += int(math.Abs(float64(floor - localElevator.CurrentFloor)))

			if cost < bestCost {
				bestCost = cost
				closestOrder.Update(floor, orderType, motorDirection, hallCallDirection)
			}
		}
	}

	return closestOrder
}
