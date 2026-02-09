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
	err1 := e.SetAction(action1)
	assert.Equal(t, nil, err1, "Action should NOT have produced error")
	assert.Equal(t, BMoving, e.Behavior, "Elevator should be set to moving")
	assert.Equal(t, elevio.MDUp, e.Dir, "Elevator should be set motor direction up")

	// CASE 2: Set illegal action
	action2 := Action{BMoving, 100}
	err2 := e.SetAction(action2)
	assert.Error(t, err2, "Action should have produced error")
}

func TestStop(t *testing.T) {
	e := setupTestElevator()
	e.Stop()
	assert.Equal(t, elevio.MDStop, e.Dir, "Elevator should have stop")
	assert.Equal(t, BIdle, e.Behavior, "Elevator should be idle")
}
