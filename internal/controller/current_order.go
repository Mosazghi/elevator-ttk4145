package controller

import (
	"log/slog"
	"time"

	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
)

func (order *CurrentOrder) Complete(worldView *statesync.Worldview) error {
	slog.Debug("completing order", "floor", order.Floor, "dir", order.MotorDirection, "type", order.Type)
	var err error

	if order.Type == elevio.Cab {
		err = worldView.SetCabCall(order.Floor, false)
	} else {
		time.Sleep(500 * time.Millisecond) // Ensure other nodes have time to process the hall call before completing it
		err = worldView.CompleteHallCall(order.Floor, order.HallCallDirection)
	}

	return err
}

func (order *CurrentOrder) Empty() bool {
	return order.Floor == -1
}

func (order *CurrentOrder) OppositeDirection(elevatorDirection elevio.MotorDirection) bool {
	return order.MotorDirection != elevatorDirection
}

func (order *CurrentOrder) AtFloor(floor int) bool {
	if floor == -1 {
		return false
	}
	return order.Floor == floor
}

func (order *CurrentOrder) Update(floor int, orderType elevio.ButtonType, Motordirection elevio.MotorDirection, hallCallDirection statesync.HallCallDirection) {
	order.Floor = floor
	order.Type = orderType
	order.MotorDirection = Motordirection
	order.HallCallDirection = hallCallDirection
}

func NewOrder() CurrentOrder {
	return CurrentOrder{
		Floor:             -1,
		Type:              elevio.Cab,
		MotorDirection:    elevio.MDStop,
		HallCallDirection: statesync.HallCallDirectionDown,
	}
}
