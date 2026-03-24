package controller

import (
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
)

// A ControllerTriggerSrc is a single value representing where a
// trigger is arrived from.
type ControllerTriggerSrc int

const (
	CTSFArrivalFloor ControllerTriggerSrc = iota
	CTSOrderUpdate
	CTSHCLights
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
