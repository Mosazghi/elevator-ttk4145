package controller

import (
	"log/slog"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type ControllerTriggerSrc int

const (
	CTSFArrivalFloor ControllerTriggerSrc = iota
	CTSOrderUpdate
	CTSHCLights
)

type CurrentOrder struct {
	Floor             int
	Type              elevio.ButtonType
	MotorDirection    elevio.MotorDirection
	HallCallDirection statesync.HallCallDir
}

func (order *CurrentOrder) Complete(worldView *statesync.Worldview) {
	if order.Type == elevio.Cab {
		err := worldView.SetCabCall(order.Floor, false)
		if err != nil {
			slog.Error("error completing cab call", "err", err)
		}
	} else {
		err := worldView.CompleteHallCall(order.Floor, order.HallCallDirection)
		if err != nil {
			slog.Error("error completing hall call", "err", err)
		}
	}
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

func (order *CurrentOrder) Update(floor int, orderType elevio.ButtonType, Motordirection elevio.MotorDirection, hallCallDirection statesync.HallCallDir) {
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
		HallCallDirection: statesync.HDDown,
	}
}
