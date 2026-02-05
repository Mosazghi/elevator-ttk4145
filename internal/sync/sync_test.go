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

	// wv2 is empty (no elevator states, no hall calls)
	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)

	require.NoError(t, err)
}

// Test 5: Merge with complex nested data (multiple elevators)
func TestMerge_MultipleElevators_ShouldSucceed(t *testing.T) {
	wv1 := NewTestWorldview(1, 4)
	wv2 := NewTestWorldview(2, 4)

	// Add multiple elevator states with different states
	state1 := NewRemoteElevatorState(1, 0, 4)
	state1.Direction = elevio.MDUp
	state1.Behavior = elevator.BMoving
	state1.CabCalls[2] = true
	state1.CabCalls[3] = true
	wv2.elevatorStates[1] = state1

	state2 := NewRemoteElevatorState(2, 3, 4)
	state2.DoorState = elevator.DSOpen
	state2.Behavior = elevator.BDoorOpen
	wv2.elevatorStates[2] = state2

	state3 := NewRemoteElevatorState(3, 1, 4)
	state3.Direction = elevio.MDDown
	state3.Behavior = elevator.BMoving
	state3.CabCalls[0] = true
	wv2.elevatorStates[3] = state3

	// Add all hall calls
	for floor := 0; floor < 4; floor++ {
		wv2.hallCalls[floor] = HallCallPair{
			Up: HallCallPairState{
				State: func() HallCallState {
					if floor < 3 {
						return HSAvailable
					}
					return HSNone
				}(),
				By: 2,
			},
			Down: HallCallPairState{
				State: func() HallCallState {
					if floor > 0 {
						return HSAvailable
					}
					return HSNone
				}(),
				By: 2,
			},
		}
	}
>>>>>>> 0ce3d3b (feat: interface)

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
// Test 8: Merge with all motor directions
func TestMerge_AllDirections_ShouldSucceed(t *testing.T) {
	directions := []elevio.MotorDirection{
		elevio.MDStop,
		elevio.MDUp,
		elevio.MDDown,
	}

	for i, dir := range directions {
		t.Run(dir.String(), func(t *testing.T) {
			wv1 := NewTestWorldview(1, 4)
			wv2 := NewTestWorldview(2, 4)

			state := NewRemoteElevatorState(i+200, 1, 4)
			state.Direction = dir
			wv2.elevatorStates[i+200] = state

			require.NoError(t, wv2.updateChecksum())

			err := wv1.Merge(wv2)
			require.NoError(t, err)
		})
	}
}

// Test 9: Merge with all behaviors
func TestMerge_AllBehaviors_ShouldSucceed(t *testing.T) {
	behaviors := []elevator.Behavior{
		elevator.BIdle,
		elevator.BDoorOpen,
		elevator.BMoving,
	}

	for i, behavior := range behaviors {
		t.Run(behavior.String(), func(t *testing.T) {
			wv1 := NewTestWorldview(1, 4)
			wv2 := NewTestWorldview(2, 4)

			state := NewRemoteElevatorState(i+300, 1, 4)
			state.Behavior = behavior
			wv2.elevatorStates[i+300] = state

			require.NoError(t, wv2.updateChecksum())

			err := wv1.Merge(wv2)
			require.NoError(t, err)
		})
	}
}

// Test 10: Merge with various cab call patterns
func TestMerge_CabCallPatterns_ShouldSucceed(t *testing.T) {
	testCases := []struct {
		name     string
		cabCalls []bool
	}{
		{"no calls", []bool{false, false, false, false}},
		{"all calls", []bool{true, true, true, true}},
		{"odd floors", []bool{false, true, false, true}},
		{"even floors", []bool{true, false, true, false}},
		{"top only", []bool{false, false, false, true}},
		{"bottom only", []bool{true, false, false, false}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wv1 := NewTestWorldview(1, 4)
			wv2 := NewTestWorldview(2, 4)

			state := NewRemoteElevatorState(400, 1, 4)
			state.CabCalls = tc.cabCalls
			wv2.elevatorStates[400] = state

			require.NoError(t, wv2.updateChecksum())

			err := wv1.Merge(wv2)
			require.NoError(t, err)
		})
	}
}

// Test 11: Merge with timestamp variations
func TestMerge_Timestamps_ShouldSucceed(t *testing.T) {
	wv1 := NewTestWorldview(1, 4)
	wv2 := NewTestWorldview(2, 4)

	now := time.Now()

	// Old timestamp
	state1 := NewRemoteElevatorState(1, 0, 4)
	state1.LastSeenAt = now.Add(-5 * time.Minute)
	wv2.elevatorStates[1] = state1

	// Current timestamp
	state2 := NewRemoteElevatorState(2, 1, 4)
	state2.LastSeenAt = now
	wv2.elevatorStates[2] = state2

	// Future timestamp (clock skew)
	state3 := NewRemoteElevatorState(3, 2, 4)
	state3.LastSeenAt = now.Add(2 * time.Second)
	wv2.elevatorStates[3] = state3

	require.NoError(t, wv2.updateChecksum())

	err := wv1.Merge(wv2)
	require.NoError(t, err)
}

// Test 12: Concurrent merge operations (thread safety)
func TestMerge_Concurrent_ShouldNotRace(t *testing.T) {
	wv1 := NewTestWorldview(1, 4)

	var wg sync.WaitGroup

	// Launch 20 concurrent merge operations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			wv := NewTestWorldview(id, 4)
			state := NewRemoteElevatorState(id, id%4, 4)
			state.CabCalls[id%4] = true
			wv.elevatorStates[id] = state
			_ = wv.updateChecksum()

			_ = wv1.Merge(wv)
		}(i + 2)
	}

	wg.Wait()
	// If we get here without panic, thread safety works
}

// Test 13: Merge nil worldview
>>>>>>> 0ce3d3b (feat: interface)
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
		Direction:    elevio.MDUp,
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
