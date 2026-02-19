package elevator

import (
	"testing"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/stretchr/testify/assert"
)

func setupTestElevator() ElevatorState {
	elevIoDriver := elevio.NewElevIoFakeDriver(4)
	return NewElevator(BIdle, elevio.MDUp, elevIoDriver)
}

func TestSetAction(t *testing.T) {
	e := setupTestElevator()

	// CASE 1: Set action
	action1 := Action{BMoving, elevio.MDUp}
	e.SetAction(action1)
	assert.Equal(t, BMoving, e.Behavior, "Elevator should be set to moving")
	assert.Equal(t, elevio.MDUp, e.Dir, "Elevator should be set motor direction up")

	// CASE 2: Set illegal action
	prevState := e
	action2 := Action{BMoving, 100}
	e.SetAction(action2)
	assert.Equal(t, prevState.Behavior, e.Behavior, "Elevator should be set to moving")
	assert.Equal(t, prevState.Dir, e.Dir, "Elevator should be set motor direction up")
}
