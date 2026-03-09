package elevator

import elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"

type MoveAction struct {
	Behavior  Behavior
	Direction elevio.MotorDirection
}

type (
	StopAction  struct{}
	LightAction struct {
		ButtonType elevio.ButtonType
		Floor      int
		State      bool
	}
)

type DoorAction struct {
	Open bool
}

type ClearOrdersAction struct {
	Floor int
}
