package statesync

import (
	"testing"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Merge with different number of floors should fail
func TestMerge_DifferentNumFloors_ShouldError(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	wv2 := NewWorldView(2, 3)

	err := wv1.Merge(wv2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesnt match")
}

// Merge with checksum mismatch should fail
func TestMerge_ChecksumMismatch_ShouldError(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	wv2 := NewWorldView(2, 4)

	// Corrupt the checksum
	wv2.checksum = 0xDEADBEEF

	err := wv1.Merge(wv2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

// Valid merge with same numFloors and valid checksum
func TestMerge_ValidInput_ShouldSucceed(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	wv2 := NewWorldView(2, 4)

	// Add elevator state to wv2
	wv2.elevatorStates[2] = NewRemoteElevatorState(2, 2, 4)
	wv2.hallCalls[1][HDUp] = HallCallPairState{
		State: HSAvailable, By: 2,
	}
	wv2.hallCalls[1][HDDown] = HallCallPairState{
		State: HSNone, By: 0,
	}

	require.NoError(t, wv2.updateChecksum())
	err := wv1.Merge(wv2)

	require.NoError(t, err)

	assert.Equal(t, wv1.hallCalls[1][HDUp].State, HSAvailable, "hall call from wv2 should be merged into wv1")
	assert.Equal(t, wv1.hallCalls[1][HDUp].By, 2, "hall call from wv2 should be merged into wv1")
}

// Merge with empty worldview
func TestMerge_EmptyWorldview_ShouldSucceed(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	wv2 := NewWorldView(2, 4)

	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)

	require.NoError(t, err)
}

// Merge with elevator at different floors
func TestMerge_ElevatorPositions_ShouldSucceed(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	wv2 := NewWorldView(2, 4)

	// Test all valid floor positions
	for floor := 0; floor < 4; floor++ {
		elevatorID := floor + 1
		state := NewRemoteElevatorState(elevatorID, floor, 4)
		wv2.elevatorStates[elevatorID] = state
	}

	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)

	require.NoError(t, err)

	// verify that only the receiving elevator is merged and not the other ones
	wv2ID := 2
	assert.Contains(t, wv1.elevatorStates, wv2ID, "elevator %d should be in wv1", wv2ID)
	for floor := 0; floor < 4; floor++ {
		elevatorID := floor + 1
		if elevatorID == wv2ID {
			continue // already checked
		}
		_, exists := wv1.elevatorStates[elevatorID]
		assert.False(t, exists, "elevator %d should not be in merged worldview", elevatorID)
	}
}

// Test merge nil worldview
func TestMerge_NilWorldview_ShouldError(t *testing.T) {
	wv1 := NewWorldView(1, 4)

	err := wv1.Merge(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// Test merge preserves local state
func TestMerge_PreservesLocalState(t *testing.T) {
	wv1 := NewWorldView(1, 4)
	originalLocalID := wv1.localID
	originalLocalState := wv1.localRemoteState

	wv2 := NewWorldView(10, 4)
	wv2.elevatorStates[10] = NewRemoteElevatorState(10, 2, 4)
	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)
	require.NoError(t, err)

	assert.Equal(t, originalLocalID, wv1.localID, "local ID should not change")
	assert.Equal(t, originalLocalState.ID, wv1.localRemoteState.ID, "local state ID should not change")
}

// Test 15: Merge with edge case: PrevFloor and TargetFloor
func TestMerge_FloorTransitions_ShouldSucceed(t *testing.T) {
	wv2ID := 10
	wv1 := NewWorldView(1, 4)

	wv2 := NewWorldView(wv2ID, 4)

	state := RemoteElevatorState{
		ID:           wv2ID,
		TargetFloor:  3,
		PrevFloor:    1,
		CurrentFloor: 2,
		Direction:    elevio.Up,
		DoorState:    elevator.DSClosed,
		CabCalls:     []bool{false, false, false, true},
		Behavior:     elevator.BMoving,
		LastSeenAt:   time.Now(),
	}
	wv2.elevatorStates[wv2ID] = state

	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)
	require.NoError(t, err)

	// Verify floor transition fields were merged correctly
	assert.Contains(t, wv1.elevatorStates, wv2ID, "elevator should be in wv1")
	assert.Equal(t, 3, wv1.elevatorStates[wv2ID].TargetFloor, "target floor should match")
	assert.Equal(t, 1, wv1.elevatorStates[wv2ID].PrevFloor, "previous floor should match")
	assert.Equal(t, 2, wv1.elevatorStates[wv2ID].CurrentFloor, "current floor should match")
	assert.Equal(t, elevio.Up, wv1.elevatorStates[wv2ID].Direction, "direction should match")
	assert.True(t, wv1.elevatorStates[wv2ID].CabCalls[3], "cab call for floor 3 should be set")
}

// Test Hall call state machine transitions
func TestMerge_HallCallStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		ourState      HallCallState
		theirState    HallCallState
		expectedState HallCallState
		shouldChange  bool
	}{
		// Valid transitions
		{"None -> Available", HSNone, HSAvailable, HSAvailable, true},
		{"Available -> Processing", HSAvailable, HSProcessing, HSProcessing, true},
		{"Processing -> None (completed)", HSProcessing, HSNone, HSNone, true},

		// Invalid/ignored transitions
		{"Available -> Available (duplicate)", HSAvailable, HSAvailable, HSAvailable, false},
		{"Processing -> Available (cannot go back)", HSProcessing, HSAvailable, HSProcessing, false},
		{"None -> Processing (skip Available)", HSNone, HSProcessing, HSNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wv1 := NewWorldView(1, 4)
			wv2 := NewWorldView(2, 4)

			wv1.hallCalls[1][HDUp] = HallCallPairState{State: tt.ourState, By: 1}

			wv2.hallCalls[1][HDUp] = HallCallPairState{State: tt.theirState, By: 2}

			require.NoError(t, wv2.updateChecksum())

			// Merge
			err := wv1.Merge(wv2)
			require.NoError(t, err)

			// Verify transition
			assert.Equal(t, tt.expectedState, wv1.hallCalls[1][HDUp].State,
				"state transition %v -> %v should result in %v",
				tt.ourState, tt.theirState, tt.expectedState)
		})
	}
}

func TestSetHallCall_Fail(t *testing.T) {
	wv := NewWorldView(1, 4)

	err := wv.SetHallCall(3, HDUp, HSAvailable)

	assert.NoError(t, err, "should be a valid state")

	err = wv.SetHallCall(3, HDUp, HSNone)

	assert.Error(t, err, "should not be able to transition from Available to None")

	err = wv.SetHallCall(3, HDUp, HSProcessing)

	assert.NoError(t, err, "should be able to transition from Available to Processing")

	err = wv.SetHallCall(3, HDUp, HSAvailable)

	assert.Error(t, err, "should not be able to transition from Processing to Available")

	err = wv.SetHallCall(3, HDUp, HSNone)

	assert.NoError(t, err, "should be able to transition from Processing to None")
}
