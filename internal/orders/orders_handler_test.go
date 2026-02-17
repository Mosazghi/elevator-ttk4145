package orders

import (
	"fmt"
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

// func TestGetNextOrder(t *testing.T) {
// 	TestGetNextOrder_HallCall(t)
// 	TestGetNextOrder_HallCall_Complete(t)
//
// 	TestGetNextOrder_CabCall(t)
// 	TestGetNextOrder_CabCall_Complete(t)
//
// 	TestGetNextOrder_CabCall_Direction(t)
// 	// TestGetNextOrder_Two_Elevators(t) //TODO: Method to check multiple elevators
// }

// Helper
func newTestCtx() (wv *statesync.Worldview, channel chan statesync.Worldview) {
	wvChan := make(chan statesync.Worldview, 10)
	worldview := statesync.NewWorldView(1, 4, wvChan)
	elev := statesync.NewRemoteElevatorState(ID, NumFloors)
	_ = worldview.SetLocalElevator(elev)

	return worldview, wvChan
}

// CASE 1: Given a Hall-Call
func TestGetNextOrder_HallCall(t *testing.T) {
	actionChan := make(chan elevator.Action)
	wv, wvchan := newTestCtx()
	go GetNextOrder(wvchan, actionChan)

	err := wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after creating new hall call")

	select {
	case action := <-actionChan:
		hallCalls := wv.GetAllHallCalls()
		call := hallCalls[3][statesync.HDUp]

		require.Equal(t, call.By, ID, "Elevator 1 should be assigend to hall-call order")
		assert.Equal(t, action.Direction, elevio.MDUp, "Expected elevator 1 to move up from floor 1 to floor 4")
		assert.Equal(t, action.Behavior, elevator.BMoving, "Elevator 1 should attempt to move")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 2: At Hall-call order
func TestGetNextOrder_HallCall_Complete(t *testing.T) {
	actionChan := make(chan elevator.Action)
	wv, wvchan := newTestCtx()
	elev := wv.GetRemoteElevator()
	go GetNextOrder(wvchan, actionChan)

	elev.CurrentFloor = 3
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")
	err = wv.NewHallCall(3, statesync.HDUp)
	require.NoError(t, err, "Failed to create new hall call")
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after creating new hall call")

	select {
	case action := <-actionChan:
		hallCalls := wv.GetAllHallCalls()
		call := hallCalls[3][statesync.HDUp]

		require.Equal(t, call.State, statesync.HSNone, "Hall call needs to be set to none, arrived at floor")
		assert.Equal(t, action.Direction, elevio.MDStop, "Expected elevator 1 to stop at order floor")
		assert.Equal(t, action.Behavior, elevator.BIdle, "Elevator 1 should open door when arrived at order floor")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 3: Given a Cab-call
func TestGetNextOrder_CabCall(t *testing.T) {
	actionChan := make(chan elevator.Action)
	wv, wvchan := newTestCtx()
	go GetNextOrder(wvchan, actionChan)

	err := wv.SetCabCall(2, true)
	require.NoError(t, err, "Failed to set cab call")
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab call")

	select {
	case action := <-actionChan:
		elev := wv.GetRemoteElevator()

		require.Equal(t, elev.CabCalls[2], true, "Cab-call should be set to true")
		assert.Equal(t, action.Direction, elevio.MDUp, "Expected elevator 1 to move up")
		assert.Equal(t, action.Behavior, elevator.BMoving, "Elevator 1 should attempt to be move")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 4: At Cab-call order
func TestGetNextOrder_CabCall_Complete(t *testing.T) {
	actionChan := make(chan elevator.Action)
	wv, wvchan := newTestCtx()
	elev := wv.GetRemoteElevator()
	go GetNextOrder(wvchan, actionChan)

	elev.CurrentFloor = 2
	_ = wv.SetLocalElevator(&elev)
	_ = wv.SetCabCall(2, true)
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab call")

	select {
	case action := <-actionChan:
		elev := wv.GetRemoteElevator()

		require.Equal(t, elev.CabCalls[1], false, "Cab-call should be set to false, arrived at floor")
		assert.Equal(t, action.Direction, elevio.MDStop, "Expected elevator 1 to stop at order floor")
		assert.Equal(t, action.Behavior, elevator.BIdle, "Elevator 1 should open door when arrived at order floor")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 5: Moving while there are Cab-calls both above and under
// FIXME: Remember to set current floor first!
func TestGetNextOrder_CabCall_Direction(t *testing.T) {
	t.Skip()
	actionChan := make(chan elevator.Action)
	wv, wvchan := newTestCtx()
	go GetNextOrder(wvchan, actionChan)

	elev := wv.GetRemoteElevator()
	elev.Behavior = elevator.BMoving
	elev.Direction = elevio.MDDown
	err := wv.SetLocalElevator(&elev)
	require.NoError(t, err, "Failed to set local elevator")

	err = wv.SetCabCall(3, true)
	require.NoError(t, err, "Failed to set cab call")
	err = wv.SetCabCall(1, true)
	require.NoError(t, err, "Failed to set cab call")
	cs, err := checksum.CalculateChecksum(wv)
	require.NoError(t, err, "Failed to calculate checksum")
	err = wv.Merge(wv, cs)
	require.NoError(t, err, "Failed to merge worldview after setting cab calls")

	fmt.Println("Current floor:", elev.CurrentFloor)

	select {
	case action := <-actionChan:
		assert.Equal(t, action.Direction, elevio.MDDown, "Elevator 1 should move down towards lower cab-call")
		assert.Equal(t, action.Behavior, elevator.BMoving, "Elevator 1 should be moving")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for action")
	}
}

// CASE 6: Two elevators gets order and you're not supposed to perform it
// func TestGetNextOrder_Two_Elevators(t *testing.T) {
// 	wv, ctx := newTestCtx()
//
// 	otherElevator := statesync.NewRemoteElevatorState(ID+1, 3)
// 	_ = wv.SetLocalElevator(otherElevator)
// 	_ = wv.NewHallCall(4, statesync.HDUp)
//
// 	behavior, dir := ctx.GetNextOrder(ID)
// 	hallCalls := wv.GetAllHallCalls()
// 	call := hallCalls[4][statesync.HDUp]
//
// 	require.Equal(t, call.By, ID+1, "Elevator 2 should be assigend to hall-call order")
// 	assert.Equal(t, dir, elevio.MDStop, "Expected Elevator 1 to stay still")
// 	assert.Equal(t, behavior, elevator.BIdle, "Elevator 1 should be idle")
// }
