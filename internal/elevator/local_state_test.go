package elevator

import (
	"testing"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/stretchr/testify/assert"
)

func setupTestElevator(t *testing.T) ElevatorState {
	t.Helper()
	elevIoDriver := elevio.NewElevIoFakeDriver(4)
	return NewElevator(BIdle, elevio.MDUp, elevIoDriver)
}

func TestSetAction(t *testing.T) {
	e := setupTestElevator(t)

	// CASE 1: Set action
	action1 := MoveAction{BMoving, elevio.MDUp}
	e.DoMotorAction(action1)
	assert.Equal(t, BMoving, e.Behavior, "Elevator should be set to moving")
	assert.Equal(t, elevio.MDUp, e.Dir, "Elevator should be set motor direction up")

	// CASE 2: Set illegal action
	prevState := e
	action2 := MoveAction{BMoving, 100}
	e.DoMotorAction(action2)
	assert.Equal(t, prevState.Behavior, e.Behavior, "Elevator should be set to moving")
	assert.Equal(t, prevState.Dir, e.Dir, "Elevator should be set motor direction up")
}
