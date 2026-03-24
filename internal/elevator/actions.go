package elevator

import elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"

type MoveAction struct {
	Behavior  Behavior
	Direction elevio.MotorDirection
}

type StopAction struct {
	Behavior Behavior
}

type (
	LightAction struct {
		ButtonType elevio.ButtonType
		Floor      int
		State      bool
	}
)

type DoorAction struct {
	Open bool
}
