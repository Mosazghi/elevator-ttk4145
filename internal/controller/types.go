package controller

import (
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
