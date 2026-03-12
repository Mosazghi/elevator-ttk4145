package controller

import (
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// HallDirToButtonType maps statesync HallCallDir to the correct elevio ButtonType.
func HallDirToButtonType(dir statesync.HallCallDir) elevio.ButtonType {
	if dir == statesync.HDDown {
		return elevio.HallDown
	}
	return elevio.HallUp
}
