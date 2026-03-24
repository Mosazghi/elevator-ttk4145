package elevator

import elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"

// Action represents a command for the elevator to perform.

// MoveAction represents an action to move the elevator in a specific direction with a certain behavior.
type MoveAction struct {
	Behavior  Behavior
	Direction elevio.MotorDirection
}

// StopAction represents an action to stop the elevator with a specific behavior.
type StopAction struct {
	Behavior Behavior
}

// LightAction represents an action to change the state of a button light.
type LightAction struct {
	ButtonType elevio.ButtonType
	Floor      int
	State      bool
}

// DoorAction represents an action to change the state of the elevator door.
type DoorAction struct {
	Open bool
}
