package controller

import (
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
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
	HallCallDirection statesync.HallCallDirection
}
