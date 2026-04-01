package controller

import (
	"log/slog"

	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	"github.com/Mosazghi/elevator-ttk4145/pkg/shared"
)

// A CurrentOrder encapsulate both hall calls and cab calls.
//
// The MotorDirection field is based on the positioning to the
// elevator which captures the order.
type CurrentOrder struct {
	Floor             int
	Type              elevio.ButtonType
	MotorDirection    elevio.MotorDirection
	HallCallDirection statesync.HallCallDirection
}

// NewOrder creates a new instance of a CurrentOrder
func NewOrder() CurrentOrder {
	return CurrentOrder{
		Floor:             -1,
		Type:              elevio.Cab,
		MotorDirection:    elevio.MotorDirectionStop,
		HallCallDirection: statesync.HallCallDirectionDown,
	}
}

// Complete finishes an order.
//
// This function updates the world view accordingly.
func (order *CurrentOrder) Complete(worldView *statesync.Worldview) error {
	slog.Debug("completing order", "floor", order.Floor, "dir", order.MotorDirection, "type", order.Type)
	var err error

	if order.Type == elevio.Cab {
		err = worldView.SetCabCall(order.Floor, false)
	} else {
		err = worldView.CompleteHallCall(order.Floor, order.HallCallDirection)
	}

	return err
}

// Empty checks if the current order is empty
func (order *CurrentOrder) Empty() bool {
	return order.Floor == -1
}

// OppositeDirection checks if the given argument has a different direction
func (order *CurrentOrder) OppositeDirection(elevatorDirection elevio.MotorDirection) bool {
	return order.MotorDirection != elevatorDirection
}

// AtFloor checks if the given argument is at the same floor as the order
func (order *CurrentOrder) AtFloor(floor int) bool {
	if floor == shared.UndefinedFloor {
		return false
	}
	return order.Floor == floor
}

// Update overwrites the order with the given arguments
func (order *CurrentOrder) Update(floor int, orderType elevio.ButtonType, Motordirection elevio.MotorDirection, hallCallDirection statesync.HallCallDirection) {
	order.Floor = floor
	order.Type = orderType
	order.MotorDirection = Motordirection
	order.HallCallDirection = hallCallDirection
}
