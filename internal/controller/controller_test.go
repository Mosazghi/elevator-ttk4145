package controller

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ID        = 1
	NumFloors = 4
)

// Helper
func newTestCtx() (wv *statesync.Worldview, wvChan chan statesync.Worldview, arriveAtFloorChan chan struct{}) {
	wvChan = make(chan statesync.Worldview, 10)
	arriveAtFloorChan = make(chan struct{}, 10)
	worldview := statesync.NewWorldView(1, 4, wvChan, make(chan statesync.Order, 10))
	elev := statesync.NewRemoteElevatorState(ID, NumFloors)
	_ = worldview.SetLocalElevator(elev)

	return worldview, wvChan, arriveAtFloorChan
}

func newTestStart(wv *statesync.Worldview, arriveAtFloor chan struct{}, actionChan chan any, newOrder chan struct{}, hcLight chan statesync.Order) {
	go Start(wv, arriveAtFloor, actionChan, newOrder, hcLight)
}

// CASE 1: Given a Hall-Call
func TestGetNextAction_HallCall(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()

	localElevator := wv.GetRemoteElevator()
	localElevator.CurrentFloor = 0

	err := wv.SetLocalElevator(&localElevator)
	require.NoError(t, err, "Failed to set initial position of elevator")
	err = wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")

	// Assign the call to this elevator by transitioning to Processing
	err = wv.ProcessHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to process hall call")

	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after creating new hall call")

	// Start the goroutine AFTER state setup is complete
	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	newOrder <- struct{}{}
	select {
	case <-newOrder:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)
		if closestOrder.Empty() {
			t.Fatal("closestOrder was empty")
		}
		if localElevator.AllowedToServe() {
			assert.Equal(t, closestOrder.TravelDirection, elevio.MDUp, "Expected direction to be up")
		}
		require.Equal(t, closestOrder.Floor, 3, "Expected closest order to be floor 3")

	case action := <-actionChan:
		assert.Equal(t, action.(elevator.MoveAction).Direction, elevio.MDUp, "Expected elevator 1 to move up from floor 0 to floor 3")
		assert.Equal(t, action.(elevator.MoveAction).Behavior, elevator.BMoving, "Elevator 1 should attempt to move")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 2: At Hall-call order
func TestGetNextAction_HallCall_Complete(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()
	elev := wv.GetRemoteElevator()

	err := wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")

	elev.CurrentFloor = 3
	err = wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")

	// Assign the call to this elevator by transitioning to Processing
	err = wv.ProcessHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to process hall call")

	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after creating new hall call")

	// Start the goroutine AFTER state setup is complete
	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	arriveAtFloor <- struct{}{}
	select {
	case <-arriveAtFloor:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)
		if closestOrder.Empty() {
			t.Fatal("No calls available")
		}
		if closestOrder.AtFloor(localElevator.CurrentFloor) {
			ArrivalSequence(wv, actionChan)
			time.Sleep(500 * time.Millisecond)
			closestOrder.Complete(wv)
		}
		hallCalls := wv.GetAllHallCalls()
		call := hallCalls[3][statesync.HDUp]
		require.Equal(t, call.State, statesync.HSNone, "Hall call needs to be set to none, arrived at floor")

	case action := <-actionChan:
		assert.Equal(t, action.(elevator.MoveAction).Direction, elevio.MDStop, "Expected elevator 1 to stop at order floor")
		assert.Equal(t, action.(elevator.MoveAction).Behavior, elevator.BDoorOpen, "Elevator 1 should open door when arrived at order floor")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 3: Given a Cab-call
func TestGetNextAction_CabCall(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()
	elev := wv.GetRemoteElevator()

	elev.CurrentFloor = 1
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")
	err = wv.SetCabCall(2, true)
	require.NoError(t, err, "Failed to set cab call")
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab call")

	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	newOrder <- struct{}{}
	select {
	case <-newOrder:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)

		if closestOrder.Empty() {
			t.Fatal("closestOrder order was empty")
		}

		if localElevator.AllowedToServe() && closestOrder.AtFloor(localElevator.CurrentFloor) {
			ArrivalSequence(wv, actionChan)
			time.Sleep(500 * time.Millisecond)
			closestOrder.Complete(wv)
		}

		if localElevator.AllowedToServe() {
			actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: closestOrder.TravelDirection}
		}
		require.Equal(t, localElevator.CabCalls[2], true, "Cab-call should be set to true")

	case action := <-actionChan:
		switch action := action.(type) {
		case elevator.MoveAction:
			require.Equal(t, action.Behavior, elevator.BMoving, "Should have received Bmoving")
			require.Equal(t, action.Direction, elevio.MDUp, "Should have received Bmoving")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 4: At Cab-call order
func TestGetNextAction_CabCall_Complete(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()
	elev := wv.GetRemoteElevator()

	elev.CurrentFloor = 2
	_ = wv.SetLocalElevator(&elev)
	_ = wv.SetCabCall(2, true)
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab call")

	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	arriveAtFloor <- struct{}{}
	select {
	case <-arriveAtFloor:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)
		slog.Info("[FetchClosestOrder] got closestOrder", "floor", closestOrder.Floor)

		if closestOrder.Empty() {
			t.Fatal("No calls available, stopping")
		}

		if closestOrder.AtFloor(localElevator.CurrentFloor) {
			slog.Info("[atFloor] true", "floor", closestOrder.Floor, "type", closestOrder.Type, "direction", closestOrder.TravelDirection)
			ArrivalSequence(wv, actionChan)
			time.Sleep(500 * time.Millisecond)
			closestOrder.Complete(wv)
		}

		require.Equal(t, localElevator.CabCalls[2], false, "Cab-call should be set to false, arrived at floor")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 5: Cab-calls both above and under
func TestGetNextAction_CabCall_Direction(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()

	elev := wv.GetRemoteElevator()
	elev.CurrentFloor = 2
	elev.Behavior = elevator.BIdle
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")

	err = wv.SetCabCall(3, true)
	require.NoError(t, err, "Failed to set cab call")
	err = wv.SetCabCall(1, true)
	require.NoError(t, err, "Failed to set cab call")

	fmt.Println("Current floor:", elev.CurrentFloor)

	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	newOrder <- struct{}{}
	select {
	case <-newOrder:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)
		if closestOrder.Empty() {
			t.Fatal("closestOrder was empty")
		}
		if localElevator.AllowedToServe() {
			// Elevator at floor 2 should prefer the lower call at floor 1
			assert.Equal(t, closestOrder.TravelDirection, elevio.MDDown, "Should move down towards lower cab-call")
		}
		require.Equal(t, closestOrder.Floor, 1, "Expected closest order to be floor 1")

	case action := <-actionChan:
		// Elevator at floor 2 should prefer the lower call at floor 1
		assert.Equal(t, action.(elevator.MoveAction).Direction, elevio.MDDown, "Elevator 1 should move down towards lower cab-call")
		assert.Equal(t, action.(elevator.MoveAction).Behavior, elevator.BMoving, "Elevator 1 should be moving")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 6: Two elevators
func TestGetNextAction_multiElevator(t *testing.T) {
	actionChan := make(chan any, 10)
	wv, _, arriveAtFloor := newTestCtx()

	local := wv.GetRemoteElevator()
	other := statesync.NewRemoteElevatorState(2, 4)
	local.CurrentFloor = 2
	other.CurrentFloor = 1

	err := wv.SetLocalElevator(&local)
	require.NoError(t, err, "Failed to set local elevator")
	err = wv.SetOtherElevator(other, 2)
	require.NoError(t, err, "Failed to set other elevator")

	err = wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")
	err = wv.NewHallCall(0, statesync.HDDown)
	require.NoError(t, err, "Failed to create new hall call")

	// Assign calls - floor 3 goes to elevator 2 (using external assignment for testing)
	// floor 0 goes to elevator 1
	err = wv.ProcessHallCall(0, statesync.HDDown)
	require.NoError(t, err, "Failed to process hall call for elevator 1")

	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after creating new hall call")

	newOrder := make(chan struct{}, 10)
	hcLight := make(chan statesync.Order, 10)
	newTestStart(wv, arriveAtFloor, actionChan, newOrder, hcLight)

	newOrder <- struct{}{}
	select {
	case <-newOrder:
		localElevator := wv.GetRemoteElevator()
		closestOrder := FetchClosestOrder(wv)
		if closestOrder.Empty() {
			t.Fatal("closestOrder was empty")
		}
		if localElevator.AllowedToServe() {
			assert.Equal(t, closestOrder.TravelDirection, elevio.MDDown, "Should move down towards floor 0")
		}
		require.Equal(t, closestOrder.Floor, 0, "Expected closest order to be floor 0")

	case action := <-actionChan:
		assert.Equal(t, action.(elevator.MoveAction).Direction, elevio.MDDown, "Expected elevator 1 to move down towards floor 0")
		assert.Equal(t, action.(elevator.MoveAction).Behavior, elevator.BMoving, "Elevator 1 should be moving")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// Testing if we have mutliple hall calls that it chooses the closest
func TestCalculateNearestHallCall(t *testing.T) {
	t.Skip()
	wv, _, _ := newTestCtx()

	elev := wv.GetRemoteElevator()
	elev.CurrentFloor = 1
	elev.Behavior = elevator.BMoving
	elev.Direction = elevio.MDUp
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")

	err = wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")
	err = wv.NewHallCall(2, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")
	err = wv.NewHallCall(1, statesync.HDDown)
	require.NoError(t, err, "Failed to create new hall call")

	// Assign calls to this elevator
	err = wv.ProcessHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to process hall call")
	err = wv.ProcessHallCall(2, statesync.HDUp)
	require.NoError(t, err, "Failed to process hall call")
	err = wv.ProcessHallCall(1, statesync.HDDown)
	require.NoError(t, err, "Failed to process hall call")

	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab calls")

	order := FindClosestCabCall(wv)

	assert.Equal(t, 2, order.Floor, "Expected nearestCall to be on floor 2")
	assert.Equal(t, statesync.HDUp, order.HallCallDirection, "Expected the nearestCall to be upwards")
}

func TestCalculateNearestCabCall(t *testing.T) {
	wv, _, _ := newTestCtx()

	elev := wv.GetRemoteElevator()
	elev.CurrentFloor = 1
	elev.Behavior = elevator.BMoving
	elev.Direction = elevio.MDUp
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")

	err = wv.SetCabCall(3, true)
	require.NoError(t, err, "Failed to create cab call")
	err = wv.SetCabCall(2, true)
	require.NoError(t, err, "Failed to create cab call")
	err = wv.SetCabCall(0, true)
	require.NoError(t, err, "Failed to create cab call")

	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab calls")

	order := FindClosestCabCall(wv)
	assert.Equal(t, 2, order.Floor, "Expected nearestCall to be on floor 2")
}
