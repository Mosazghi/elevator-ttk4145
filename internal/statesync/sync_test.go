package statesync

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// Easier to create test worldviews with this helper function
func NewTestWorldView(localID, numFloors int) *Worldview {
	return NewWorldView(localID, numFloors, make(chan Worldview, 10), make(chan Order, 10))
}

// Merge with different number of floors should fail
func TestMerge_DifferentNumFloors_ShouldError(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 3)

	cs, _ := checksum.CalculateChecksum(wv2)
	err := wv1.Merge(wv2, cs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesnt match")
}

// Merge with checksum mismatch should fail
func TestMerge_ChecksumMismatch_ShouldError(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)
	checksum, _ := checksum.CalculateChecksum(wv2)
	// Corrupt the checksum
	wv2.LocalID = 999

	err := wv1.Merge(wv2, checksum)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

// Valid merge with same numFloors and valid checksum
func TestMerge_ValidInput_ShouldSucceed(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	// Add elevator state to wv2
	wv2.HallCalls[1][HDUp] = HallCallPairState{
		State: HSAvailable, By: 0,
	}
	wv2.HallCalls[1][HDDown] = HallCallPairState{
		State: HSNone, By: 0,
	}

	checksum, _ := checksum.CalculateChecksum(wv2)
	err := wv1.Merge(wv2, checksum)

	require.NoError(t, err)
	fmt.Println("ID: ", wv1.HallCalls[1][HDUp].By)
	assert.Equal(t, wv1.HallCalls[1][HDUp].State, HSAvailable, "hall call from wv2 should be merged into wv1")
	assert.Equal(t, wv1.HallCalls[1][HDUp].By, -1, "hall call from wv2 should be merged into wv1")
}

// Merge with empty worldview
func TestMerge_EmptyWorldview_ShouldSucceed(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	checksum, _ := checksum.CalculateChecksum(wv2)

	err := wv1.Merge(wv2, checksum)

	require.NoError(t, err)
}

// Merge with elevator at different floors
func TestMerge_ElevatorPositions_ShouldSucceed(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	// Test all valid floor positions
	for floor := range 4 {
		elevatorID := floor + 1
		state := NewRemoteElevatorState(elevatorID, 4)
		wv2.ElevatorStates[elevatorID] = state
	}

	checksum, _ := checksum.CalculateChecksum(wv2)

	err := wv1.Merge(wv2, checksum)

	require.NoError(t, err)

	// verify that only the receiving elevator is merged and not the other ones
	wv2ID := 2
	assert.Contains(t, wv1.ElevatorStates, wv2ID, "elevator %d should be in wv1", wv2ID)
	for floor := range 4 {
		elevatorID := floor + 1
		if elevatorID == wv2ID || elevatorID == wv1.LocalID {
			continue // already checked
		}
		_, exists := wv1.ElevatorStates[elevatorID]
		assert.False(t, exists, "elevator %d should not be in merged worldview", elevatorID)
	}
}

// Test merge nil worldview
func TestMerge_NilWorldview_ShouldError(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)

	err := wv1.Merge(nil, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// Test merge preserves local state
func TestMerge_PreservesLocalState(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	originalLocalID := wv1.LocalID
	originalLocalState := wv1.ElevatorStates[originalLocalID]

	wv2 := NewTestWorldView(10, 4)
	wv2.ElevatorStates[10] = NewRemoteElevatorState(10, 4)

	checksum, _ := checksum.CalculateChecksum(wv2)
	err := wv1.Merge(wv2, checksum)
	require.NoError(t, err)

	assert.Equal(t, originalLocalID, wv1.LocalID, "local ID should not change")
	assert.Equal(t, originalLocalState.ID, wv1.ElevatorStates[originalLocalID].ID, "local state ID should not change")
}

// Test 15: Merge with edge case: PrevFloor and TargetFloor
func TestMerge_FloorTransitions_ShouldSucceed(t *testing.T) {
	wv2ID := 10
	wv1 := NewTestWorldView(1, 4)

	wv2 := NewTestWorldView(wv2ID, 4)

	wv2.ElevatorStates[wv2ID] = &RemoteElevatorState{
		ID:           wv2ID,
		TargetFloor:  3,
		CurrentFloor: 2,
		Direction:    elevio.MDUp,
		DoorState:    elevator.DSClosed,
		CabCalls:     []bool{false, false, false, true},
		Behavior:     elevator.BMoving,
		LastSeenAt:   time.Now(),
		NumFloors:    4,
	}

	checksum, _ := checksum.CalculateChecksum(wv2)
	err := wv1.Merge(wv2, checksum)
	require.NoError(t, err)

	// Verify floor transition fields were merged correctly
	assert.Contains(t, wv1.ElevatorStates, wv2ID, "elevator should be in wv1")
	assert.Equal(t, 3, wv1.ElevatorStates[wv2ID].TargetFloor, "target floor should match")
	assert.Equal(t, 2, wv1.ElevatorStates[wv2ID].CurrentFloor, "current floor should match")
	assert.Equal(t, elevio.MDUp, wv1.ElevatorStates[wv2ID].Direction, "direction should match")
	assert.True(t, wv1.ElevatorStates[wv2ID].CabCalls[3], "cab call for floor 3 should be set")
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
		{"Processing -> Available (order released)", HSProcessing, HSAvailable, HSAvailable, true},
		{"None -> Processing (skip Available)", HSNone, HSProcessing, HSNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wv1 := NewTestWorldView(1, 4)
			wv2 := NewTestWorldView(2, 4)

			wv1.HallCalls[1][HDUp] = HallCallPairState{State: tt.ourState, By: 2}

			wv2.HallCalls[1][HDUp] = HallCallPairState{State: tt.theirState, By: 2}

			checksum, _ := checksum.CalculateChecksum(wv2)
			// Merge
			err := wv1.Merge(wv2, checksum)
			require.NoError(t, err)

			// Verify transition
			assert.Equal(t, tt.expectedState, wv1.HallCalls[1][HDUp].State,
				"state transition %v -> %v should result in %v",
				tt.ourState, tt.theirState, tt.expectedState)
		})
	}
}

func TestSetHallCall(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	// Test invalid floors
	err := wv.NewHallCall(-1, HDUp)
	assert.Error(t, err, "should reject negative floor")

	err = wv.NewHallCall(4, HDUp)
	assert.Error(t, err, "should reject floor > NumFloors")

	// Test boundary floors
	err = wv.NewHallCall(0, HDUp)
	assert.NoError(t, err, "should accept floor 0")

	err = wv.NewHallCall(3, HDDown)
	assert.NoError(t, err, "should accept last floor with HDDown")

	// Test None -> Available with ConfirmedBy verification
	err = wv.NewHallCall(2, HDUp)
	assert.NoError(t, err, "should be a valid state transition")
	assert.Equal(t, HSAvailable, wv.HallCalls[2][HDUp].State, "state should be Available")
	assert.Contains(t, wv.HallCalls[2][HDUp].ConfirmedBy, wv.LocalID, "ConfirmedBy should contain localID")
	assert.Equal(t, -1, wv.HallCalls[2][HDUp].By, "By should be -1 for Available state")

	// Test Available -> None (should fail without processing first)
	err = wv.CompleteHallCall(2, HDUp)
	assert.Error(t, err, "should not be able to transition from Available to None")

	// Manually assign the call to local elevator (simulates external assignment logic)

	// Test Available -> Processing with field verification
	err = wv.ProcessHallCall(2, HDUp)

	assert.NoError(t, err, "should be able to transition from Available to Processing")
	assert.Equal(t, HSProcessing, wv.HallCalls[2][HDUp].State, "state should be Processing")
	assert.Equal(t, wv.LocalID, wv.HallCalls[2][HDUp].By, "By should be localID when processing")
	assert.Contains(t, wv.HallCalls[2][HDUp].ConfirmedBy, wv.LocalID, "ConfirmedBy should be preserved")

	// Test Processing -> Available (should fail)
	err = wv.NewHallCall(2, HDUp)
	assert.Error(t, err, "should not be able to transition from Processing to Available")

	// Test Processing -> None (complete)
	err = wv.CompleteHallCall(2, HDUp)
	assert.NoError(t, err, "should be able to transition from Processing to None")
	assert.Equal(t, HSNone, wv.HallCalls[2][HDUp].State, "state should be None")
	// assert.Equal(t, -1, wv.HallCalls[2][HDUp].By, "By should be reset to -1 after completion")
	assert.Empty(t, wv.HallCalls[2][HDUp].ConfirmedBy, "ConfirmedBy should be empty after completion")
}

func TestSetCabCall(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	err := wv.SetCabCall(2, true)

	assert.NoError(t, err, "should be able to set valid cab call")
	assert.True(t, wv.ElevatorStates[wv.LocalID].CabCalls[2], "cab call state should be updated")

	err = wv.SetCabCall(2, false)

	assert.NoError(t, err, "should be able to set valid cab call")
	assert.False(t, wv.ElevatorStates[wv.LocalID].CabCalls[2], "cab call state should be updated")

	err = wv.SetCabCall(5, true)

	assert.Error(t, err, "should not be able to set cab call for invalid floor")
}

func TestSetLocalElevator(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	validState := NewRemoteElevatorState(1, 4)

	err := wv.SetLocalElevator(validState)

	assert.NoError(t, err, "should be able to set valid local elevator state")

	invalidState := RemoteElevatorState{
		ID:           1,
		TargetFloor:  5, // invalid floor
		CurrentFloor: 2,
		Direction:    elevio.MDUp,
		DoorState:    elevator.DSOpen,
		CabCalls:     []bool{false, false, false, false},
		Behavior:     elevator.BMoving,
		LastSeenAt:   time.Now(),
	}

	err = wv.SetLocalElevator(&invalidState)

	assert.Error(t, err, "should not be able to set invalid local elevator state")
}

// TestStartSyncing_BroadcastsOwnState verifies that the worldview broadcasts its state periodically
func TestStartSyncing_BroadcastsOwnState(t *testing.T) {
	wv := NewTestWorldView(1, 4)
	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	// Run StartSyncing in background
	go wv.StartSyncing(txChan, rxChan, errChan)

	// Wait for at least one broadcast
	select {
	case msg := <-txChan:
		assert.NotNil(t, msg.Data, "broadcast should contain data")
		assert.Greater(t, len(msg.Data), 0, "broadcast data should not be empty")

		// Verify the message can be unmarshaled
		var received Message
		err := msgpack.Unmarshal(msg.Data, &received)
		require.NoError(t, err, "broadcast data should be valid msgpack")
		assert.Equal(t, wv.LocalID, received.Wv.LocalID, "broadcast should contain local ID")

	case <-time.After(BroadcastInterval * 2):
		t.Fatal("no broadcast received within expected time")
	}
}

// TestStartSyncing_ReceivesAndMergesPeerData verifies that incoming peer data is merged
func TestStartSyncing_ReceivesAndMergesPeerData(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	// Start syncing for wv1
	go wv1.StartSyncing(txChan, rxChan, errChan)

	// Set a hall call in wv2
	wv2.NewHallCall(2, HDUp)

	// Create and send a message from wv2
	jsonData, err := BuildWvJSON(wv2)
	require.NoError(t, err)

	rxChan <- network.UDPMessage{Data: jsonData}

	// Wait for merge to complete and verify
	time.Sleep(100 * time.Millisecond)

	wv1.mu.Lock()
	assert.Equal(t, HSAvailable, wv1.HallCalls[2][HDUp].State, "hall call from wv2 should be merged")
	assert.Contains(t, wv1.ElevatorStates, 2, "wv2's elevator state should be merged")
	wv1.mu.Unlock()
}

// TestStartSyncing_IgnoresOwnBroadcast verifies that we don't merge our own broadcasts
func TestStartSyncing_IgnoresOwnBroadcast(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	go wv.StartSyncing(txChan, rxChan, errChan)

	// Set original state
	originalFloor := 2
	wv.NewHallCall(originalFloor, HDUp)

	// Create a message with the same LocalID but different state
	fakeOwnMessage := NewTestWorldView(1, 4) // Same ID as wv
	fakeOwnMessage.NewHallCall(3, HDDown)

	jsonData, err := BuildWvJSON(fakeOwnMessage)
	require.NoError(t, err)

	// Send the "own" message
	rxChan <- network.UDPMessage{Data: jsonData}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Verify state was NOT changed by the ignored message
	wv.mu.Lock()
	assert.Equal(t, HSAvailable, wv.HallCalls[originalFloor][HDUp].State, "original state should remain")
	assert.Equal(t, HSNone, wv.HallCalls[3][HDDown].State, "should not merge own broadcast")
	wv.mu.Unlock()
}

// TestStartSyncing_HandlesInvalidJSON verifies graceful handling of malformed messages
func TestStartSyncing_HandlesInvalidJSON(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	go wv.StartSyncing(txChan, rxChan, errChan)

	// Send invalid JSON
	rxChan <- network.UDPMessage{Data: []byte("not valid json")}

	// Should not crash; wait and verify system still works
	time.Sleep(100 * time.Millisecond)

	// Send valid message to confirm system is still responsive
	wv2 := NewTestWorldView(2, 4)
	jsonData, err := BuildWvJSON(wv2)
	require.NoError(t, err)

	rxChan <- network.UDPMessage{Data: jsonData}
	time.Sleep(100 * time.Millisecond)

	// Verify valid message was processed
	wv.mu.Lock()
	assert.Contains(t, wv.ElevatorStates, 2, "should still process valid messages after invalid one")
	wv.mu.Unlock()
}

// TestStartSyncing_DetectsLostPeers verifies timeout detection for lost peers
func TestStartSyncing_DetectsLostPeers(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	jsonData, err := BuildWvJSON(wv2)
	require.NoError(t, err)

	go wv1.StartSyncing(txChan, rxChan, errChan)

	// Send initial message from wv2
	rxChan <- network.UDPMessage{Data: jsonData}
	time.Sleep(100 * time.Millisecond)

	// Verify peer exists
	wv1.mu.Lock()
	peer, exists := wv1.ElevatorStates[2]
	if exists {
		peer.LastSeenAt = time.Now().Add(-NodeLostTimeout - time.Second)
	}
	wv1.mu.Unlock()

	assert.True(t, exists, "peer should be added")

	// Wait for timeout detection
	time.Sleep(BroadcastInterval + 200*time.Millisecond)

	// Copy state safely
	wv1.mu.RLock()
	elev, exists := wv1.ElevatorStates[2]

	var elevCopy RemoteElevatorState
	if exists {
		elevCopy = *elev
	}
	wv1.mu.RUnlock()

	assert.True(t, exists, "peer should still exist in ElevatorStates")
	assert.False(t, elevCopy.Alive, "peer should be marked as not alive (timed out)")
}

// TestStartSyncing_ReappearedPeers verifies handling of returning peers
func TestStartSyncing_ReappearedPeers(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	// Start syncing goroutine
	go wv1.StartSyncing(txChan, rxChan, errChan)

	// Add wv2 to wv1 and mark it as "lost"
	wv1.mu.Lock()
	wv1.ElevatorStates[wv2.LocalID] = wv2.ElevatorStates[wv2.LocalID]
	wv1.ElevatorStates[wv2.LocalID].Alive = false
	wv1.mu.Unlock()

	// Ensure peer is initially marked dead
	wv1.mu.RLock()
	assert.False(t, wv1.ElevatorStates[wv2.LocalID].Alive, "peer should start as dead")
	wv1.mu.RUnlock()

	// Simulate message from the "reappeared" peer
	data, err := BuildWvJSON(wv2)
	require.NoError(t, err)
	rxChan <- network.UDPMessage{Data: data}

	// Wait for StartSyncing to process the message
	time.Sleep(100 * time.Millisecond)

	// Verify peer is now marked alive
	wv1.mu.RLock()
	peer, exists := wv1.ElevatorStates[wv2.LocalID]
	var peerCopy RemoteElevatorState
	if exists {
		peerCopy = *peer
	}
	wv1.mu.RUnlock()

	require.True(t, exists, "peer should exist in active elevators map")
	assert.True(t, peerCopy.Alive, "peer should be marked alive after reappearing")
}

// TestStartSyncing_ConcurrentAccess verifies thread safety during syncing
func TestStartSyncing_ConcurrentAccess(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	go wv.StartSyncing(txChan, rxChan, errChan)

	// Drain orderUpdateChan so NewHallCall never blocks
	go func() {
		for range wv.orderUpdateChan {
		}
	}()

	// Concurrent operations
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 50; i++ {
			wv.NewHallCall(i%4, HDUp)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Sender goroutine (simulating incoming messages)
	go func() {
		for i := 2; i < 10; i++ {
			peer := NewTestWorldView(i, 4)
			jsonData, err := BuildWvJSON(peer)
			require.NoError(t, err)
			rxChan <- network.UDPMessage{Data: jsonData}
			time.Sleep(15 * time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 50; i++ {
			_ = wv.GetAllHallCalls()
			_ = wv.GetRemoteElevator()
			time.Sleep(8 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we got here without deadlock or race condition, test passes
	assert.True(t, true, "concurrent access should not cause issues")
}

// TestStartSyncing_NetworkErrors verifies error channel handling
// TestLostNode_ReleasesPendingOrders verifies that hall calls assigned to a timed-out node are reset
func TestLostNode_ReleasesPendingOrders(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	// --- Test: lost node ---
	lostID := 2
	wv.ElevatorStates[lostID] = NewRemoteElevatorState(lostID, 4)
	// Simulate node timeout
	wv.ElevatorStates[lostID].LastSeenAt = time.Now().Add(-NodeLostTimeout - time.Second)
	wv.HallCalls[1][HDUp] = HallCallPairState{
		State:       HSProcessing,
		By:          lostID,
		ConfirmedBy: []int{lostID},
		Timestamp:   time.Now().UnixMilli(),
	}

	wv.mu.Lock()
	wv.releaseAnyOrders()
	wv.mu.Unlock()

	hc := wv.HallCalls[1][HDUp]
	assert.Equal(t, HSAvailable, hc.State, "order should be released when node is lost")
	assert.Equal(t, -1, hc.By, "By should be reset to -1")
	assert.NotContains(t, hc.ConfirmedBy, lostID, "ConfirmedBy should not include lost node")
	assert.False(t, wv.ElevatorStates[lostID].Alive, "lost node should be marked dead")

	// test for an elevator taking too long to process an order (finish)
	processingID := 1
	wv.ElevatorStates[processingID] = NewRemoteElevatorState(processingID, 4)
	wv.HallCalls[3][HDUp] = HallCallPairState{
		State:       HSProcessing,
		By:          processingID,
		ConfirmedBy: []int{processingID},
		Timestamp:   time.Now().Add(-OrderProcessingTimeout - time.Second).UnixMilli(),
	}

	wv.mu.Lock()
	wv.releaseAnyOrders()
	wv.mu.Unlock()

	hc = wv.HallCalls[3][HDUp]
	assert.Equal(t, HSAvailable, hc.State, "order should be released when processing timeout is exceeded")
	assert.Equal(t, -1, hc.By, "By should be reset to -1")
	assert.Contains(t, hc.ConfirmedBy, processingID, "ConfirmedBy should still include local node")
}

func TestStartSyncing_NetworkErrors(t *testing.T) {
	wv := NewTestWorldView(1, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	go wv.StartSyncing(txChan, rxChan, errChan)

	// Send a network error
	errChan <- fmt.Errorf("simulated network error")

	// System should continue working despite error
	time.Sleep(100 * time.Millisecond)

	// Send valid message to verify system is still functional
	wv2 := NewTestWorldView(2, 4)
	jsonData, err := BuildWvJSON(wv2)
	require.NoError(t, err)

	rxChan <- network.UDPMessage{Data: jsonData}
	time.Sleep(100 * time.Millisecond)

	// Verify system still processes messages
	wv.mu.Lock()
	_, exists := wv.ElevatorStates[2]
	wv.mu.Unlock()

	assert.True(t, exists, "should continue operating after network error")
}

// TestStartSyncing_ConfirmationMerging verifies ConfirmedBy lists are properly merged
func TestStartSyncing_ConfirmationMerging(t *testing.T) {
	wv1 := NewTestWorldView(1, 4)
	wv2 := NewTestWorldView(2, 4)
	wv3 := NewTestWorldView(3, 4)

	txChan := make(chan network.UDPMessage, 10)
	rxChan := make(chan network.UDPMessage, 10)
	errChan := make(chan error, 10)

	// fill elevator states for wv1
	for i := 1; i <= 3; i++ {
		wv1.ElevatorStates[i] = NewRemoteElevatorState(i, 4)
	}

	go wv1.StartSyncing(txChan, rxChan, errChan)

	// wv2 has a hall call confirmed by elevator 2 and 3
	wv2.HallCalls[1][HDUp] = HallCallPairState{
		State:       HSAvailable,
		By:          2,
		ConfirmedBy: []int{2, 3},
	}

	// Send wv2's state to wv1
	jsonData, err := BuildWvJSON(wv2)
	require.NoError(t, err)

	rxChan <- network.UDPMessage{Data: jsonData}
	time.Sleep(100 * time.Millisecond)

	// Verify wv1 merged the confirmations and added its own
	wv1.mu.Lock()
	confirmations := wv1.HallCalls[1][HDUp].ConfirmedBy
	wv1.mu.Unlock()

	assert.Contains(t, confirmations, 1, "should add own ID to confirmations")
	assert.Contains(t, confirmations, 2, "should merge confirmation from wv2")
	assert.Contains(t, confirmations, 3, "should merge confirmation from wv3")
	assert.Equal(t, HSAvailable, wv1.HallCalls[1][HDUp].State, "state should be Available")
	wv1.mu.Lock()
	wv1.ElevatorStates[4] = NewRemoteElevatorState(4, 4) // Add another elevator to wv1 to test merging new confirmation
	wv1.mu.Unlock()

	// Now send wv3's state with more confirmations
	wv3.HallCalls[1][HDUp] = HallCallPairState{
		State:       HSAvailable,
		By:          2,
		ConfirmedBy: []int{2, 3, 4},
	}

	jsonData2, err := BuildWvJSON(wv3)
	require.NoError(t, err)

	rxChan <- network.UDPMessage{Data: jsonData2}
	time.Sleep(100 * time.Millisecond)

	// Verify additional confirmation was merged
	wv1.mu.Lock()
	confirmations = wv1.HallCalls[1][HDUp].ConfirmedBy
	wv1.mu.Unlock()

	assert.Contains(t, confirmations, 4, "should merge new confirmation")
	assert.Len(t, confirmations, 4, "should have all unique confirmations")
}

func TestCabCallRecoveredOnReconnect(t *testing.T) {

	// Elevator A before crash
	A := NewTestWorldView(1, 4)
	A.ElevatorStates[A.LocalID].CabCalls[1] = true

	// Elevator B sees A and stores that state
	B := NewTestWorldView(2, 4)

	stateOfA := A.ElevatorStates[A.LocalID]
	B.ElevatorStates[A.LocalID] = stateOfA

	require.True(t,
		B.ElevatorStates[A.LocalID].CabCalls[1],
		"B must remember A's cab call before crash",
	)

	// --- A crashes and reboots ---
	ARebooted := NewTestWorldView(1, 4)

	assert.False(t,
		ARebooted.ElevatorStates[A.LocalID].CabCalls[1],
		"rebooted elevator should start with empty cab calls",
	)

	// --- simulate worldview exchange ---
	// simulate A being considered dead
	// A previously knew about B but thought it was dead
	ARebooted.ElevatorStates[B.LocalID] = NewRemoteElevatorState(B.LocalID, 4)
	ARebooted.ElevatorStates[B.LocalID].Alive = false
	cs, err := checksum.CalculateChecksum(B)
	require.NoError(t, err)
	require.True(t, B.ElevatorStates[1].CabCalls[1])
	ARebooted.Merge(B, cs)

	// After merge, A should recover its cab call from B
	assert.True(t,
		ARebooted.ElevatorStates[A.LocalID].CabCalls[1],
		"cab call should be recovered from peer worldview",
	)
}

func TestElevatorReconnectRecoversCabCalls(t *testing.T) {
	// --- Step 1: Elevator A has a cab call ---
	A := NewTestWorldView(1, 4)
	A.ElevatorStates[A.LocalID].CabCalls[1] = true

	// --- Step 2: Elevator B sees A's state ---
	B := NewTestWorldView(2, 4)
	B.ElevatorStates[A.LocalID] = A.ElevatorStates[A.LocalID]

	require.True(t, B.ElevatorStates[A.LocalID].CabCalls[1], "B should remember A's cab call")

	// --- Step 3: Elevator A crashes and reboots ---
	ARebooted := NewTestWorldView(1, 4)
	assert.False(t, ARebooted.ElevatorStates[A.LocalID].CabCalls[1],
		"rebooted elevator should start with empty cab calls",
	)

	// --- Step 4: Simulate B marking A as timed out ---
	B.ElevatorStates[A.LocalID].Alive = false
	B.ElevatorStates[A.LocalID].LastSeenAt = time.Now().Add(-NodeLostTimeout - time.Second)

	// Calculate checksum for worldview exchange
	cs, err := checksum.CalculateChecksum(B)
	require.NoError(t, err)

	// --- Step 5: Elevator A merges worldview from B (reconnect) ---
	err = ARebooted.Merge(B, cs)
	require.NoError(t, err)

	// --- Assertions: A should recover cab calls and ARebooted should see A as alive ---
	aState := ARebooted.ElevatorStates[A.LocalID]
	assert.True(t, aState.CabCalls[1], "cab call should be recovered from peer")
	assert.True(t, aState.Alive, "reconnected elevator should be marked alive")
}
