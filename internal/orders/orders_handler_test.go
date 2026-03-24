package orders

import (
	"testing"

	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"
	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numFloors = 4

// buildWV creates a minimal Worldview with the given elevator states for cost tests.
func buildWV(t *testing.T, localID int, elevators map[int]*statesync.RemoteElevatorState) statesync.Worldview {
	t.Helper()
	wv := statesync.NewWorldView(localID, numFloors, make(chan statesync.Order, 1), make(chan Empty, 1))
	wv.ElevatorStates = elevators
	return *wv
}

// TestCalculateCost_SingleElevator – only one elevator, it must win.
func TestCalculateCost_SingleElevator(t *testing.T) {
	elev := statesync.NewRemoteElevatorState(1, numFloors)
	elev.CurrentFloor = 0
	elev.Direction = elevio.MDUp

	wv := buildWV(t, 1, map[int]*statesync.RemoteElevatorState{1: elev})
	err := wv.NewHallCall(2, statesync.HDUp)
	require.NoError(t, err)
	err = wv.ProcessHallCall(2, statesync.HDUp)
	require.NoError(t, err)

	winner := CalculateCost(&wv, 2, statesync.HDUp)

	assert.Equal(t, 1, winner.id)
}

// TestCalculateCost_CloserElevatorWins – elevator on floor 3 beats one on floor 0 for a call on floor 3.
func TestCalculateCost_CloserElevatorWins(t *testing.T) {
	far := statesync.NewRemoteElevatorState(1, numFloors)
	far.CurrentFloor = 0

	near := statesync.NewRemoteElevatorState(2, numFloors)
	near.CurrentFloor = 3

	wv := buildWV(t, 1, map[int]*statesync.RemoteElevatorState{1: far, 2: near})
	err := wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err)
	wv.HallCalls[3][statesync.HDUp].State = statesync.HallCallStateConfirmed // Simulate assignment to trigger cost calculation
	err = wv.ProcessHallCall(3, statesync.HDUp)
	require.NoError(t, err)
	winner := CalculateCost(&wv, 3, statesync.HDUp)

	assert.Equal(t, 2, winner.id, "closer elevator should win")
}

// TestCalculateCost_ObstructedPenalty – obstructed elevator loses to an unobstructed one at equal distance.
func TestCalculateCost_ObstructedPenalty(t *testing.T) {
	obstructed := statesync.NewRemoteElevatorState(1, numFloors)
	obstructed.CurrentFloor = 2
	obstructed.IsObstructed = true

	clear := statesync.NewRemoteElevatorState(2, numFloors)
	clear.CurrentFloor = 2
	clear.IsObstructed = false

	wv := buildWV(t, 1, map[int]*statesync.RemoteElevatorState{1: obstructed, 2: clear})
	err := wv.NewHallCall(2, statesync.HDUp)
	require.NoError(t, err)
	wv.HallCalls[2][statesync.HDUp].State = statesync.HallCallStateConfirmed // Simulate assignment to trigger cost calculation
	err = wv.ProcessHallCall(2, statesync.HDUp)
	require.NoError(t, err)
	winner := CalculateCost(&wv, 2, statesync.HDUp)

	assert.Equal(t, 2, winner.id, "non-obstructed elevator should win")
}

// TestCalculateCost_WrongDirectionPenalty – elevator going down is penalised for an Up call.
func TestCalculateCost_WrongDirectionPenalty(t *testing.T) {
	goingDown := statesync.NewRemoteElevatorState(1, numFloors)
	goingDown.CurrentFloor = 2
	goingDown.Direction = elevio.MDDown

	idle := statesync.NewRemoteElevatorState(2, numFloors)
	idle.CurrentFloor = 2
	idle.Direction = elevio.MDStop

	wv := buildWV(t, 1, map[int]*statesync.RemoteElevatorState{1: goingDown, 2: idle})
	err := wv.NewHallCall(2, statesync.HDUp)
	require.NoError(t, err)
	wv.HallCalls[2][statesync.HDUp].State = statesync.HallCallStateConfirmed // Simulate assignment to trigger cost calculation
	err = wv.ProcessHallCall(2, statesync.HDUp)
	require.NoError(t, err)
	winner := CalculateCost(&wv, 2, statesync.HDUp)

	assert.Equal(t, 2, winner.id, "idle elevator should beat one going the wrong way")
}
