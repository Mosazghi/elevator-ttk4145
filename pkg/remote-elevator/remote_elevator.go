package remoteelevator

import (
	"time"

	elevator "github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
)

type RemoteElevatorState struct {
	ID           int
	TargetFloor  int
	PrevFloor    int
	CurrentFloor int
	Direction    elevio.MotorDirection
	DoorState    elevator.DoorState
	CabCalls     []bool
	Behavior     elevator.Behavior
	LastSeenAt   time.Time
}
