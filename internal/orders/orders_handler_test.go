package orders

import (
	"testing"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ID        = 1
	NumFloors = 4
)

func TestGetNextOrder(t *testing.T) {
	TestGetNextOrder_HallCall(t)
	TestGetNextOrder_CabCall(t)
	TestGetNextOrder_Arrival_CabCall(t)
	TestGetNextOrder_Arrival_HallCall(t)
	TestGetNextOrder_CabCall_Direction(t)
	TestGetNextOrder_Two_Elevators(t)
}

// Helper
func resetHallCalls(wv *statesync.Worldview) {
	hallCalls := wv.GetHallCalls()
	emptyHallCall := statesync.HallCallPair{}
	for i := range hallCalls {
		hallCalls[i] = emptyHallCall
	}
}

// Helper
func resetCabCalls(wv *statesync.Worldview) {
	elevators := wv.GetElevators()
	for elevatorIndex := range elevators {
		for cabsIndex := range elevators[elevatorIndex].CabCalls {
			elevators[elevatorIndex].CabCalls[cabsIndex] = false
		}
	}
}

// Helper
func newTestCtx() (wv *statesync.Worldview, ctx OrdersContext) {
	worldview := statesync.NewTestWorldview(0, NumFloors)
	state := statesync.NewRemoteElevatorState(ID, 1, NumFloors)
	wv.AddElevator(state)
	resetHallCalls(wv)
	resetCabCalls(wv)
	return worldview, OrdersContext{*worldview}
}

// CASE 1: Given a Hall-Call
func TestGetNextOrder_HallCall(t *testing.T) {
	wv, ctx := newTestCtx()

	hallCall := statesync.HallCallPair{
		Up: statesync.HallCallPairState{State: statesync.HSAvailable},
	}
	wv.AddHallCall(4, hallCall)

	behavior, dir := ctx.GetNextOrder(ID)
	hallCalls := wv.GetHallCalls()

	require.Equal(t, hallCalls[3].Up.By, ID, "Elevator 1 should be assigend to hall-call order")
	assert.Equal(t, dir, elevio.MDUp, "Expected elevator 1 to move up from floor 1 to floor 4")
	assert.Equal(t, behavior, elevator.BMoving, "Elevator 1 should attempt to move")
}

// CASE 2: Given a Cab-call
func TestGetNextOrder_CabCall(t *testing.T) {
	wv, ctx := newTestCtx()

	wv.SetCabCall(ID, 2, true)

	behavior, dir := ctx.GetNextOrder(ID)
	assert.Equal(t, dir, elevio.MDUp, "Expected elevator 1 to move up")
	assert.Equal(t, behavior, elevator.BMoving, "Elevator 1 should attempt to be move")
}

// CASE 3: Arrived at Cab-call floor
func TestGetNextOrder_Arrival_CabCall(t *testing.T) {
	wv, ctx := newTestCtx()

	elevators := wv.GetElevators()
	elevator1 := elevators[0]
	wv.SetCabCall(ID, 2, true)
	wv.SetElevatorPosition(ID, 3)

	behavior, dir := ctx.GetNextOrder(ID)
	require.Equal(t, elevator1.CabCalls[2], false, "Cab-call should be set to false, arrived at floor")
	assert.Equal(t, dir, elevio.MDStop, "Expected elevator 1 to stop at order floor")
	assert.Equal(t, behavior, elevator.BDoorOpen, "Elevator 1 should open door when arrived at order floor")
}

// CASE 4: Arrived at Hall-call floor
func TestGetNextOrder_Arrival_HallCall(t *testing.T) {
	wv, ctx := newTestCtx()

	hallCall := statesync.HallCallPair{
		Up: statesync.HallCallPairState{State: statesync.HSProcessing, By: ID},
	}
	wv.AddHallCall(4, hallCall)

	hallCalls := wv.GetHallCalls()
	wv.SetElevatorPosition(ID, 4)

	behavior, dir := ctx.GetNextOrder(ID)
	require.Equal(t, hallCalls[3].Up, nil, "Hall call should be set to nil, arrived at floor")
	assert.Equal(t, dir, elevio.MDStop, "Expected elevator 1 to stop at order floor")
	assert.Equal(t, behavior, elevator.BDoorOpen, "Elevator 1 should open door when arrived at order floor")
}

// CASE 5: Moving while there are Cab-calls both above and under
func TestGetNextOrder_CabCall_Direction(t *testing.T) {
	wv, ctx := newTestCtx()

	elevators := wv.GetElevators()
	elevator1 := elevators[0]
	wv.SetElevatorPosition(ID, 3)
	wv.SetCabCall(ID, 3, true)
	wv.SetCabCall(ID, 1, true)
	elevator1.Direction = elevio.MDDown

	behavior, dir := ctx.GetNextOrder(ID)
	assert.Equal(t, dir, elevio.MDDown, "Elevator 1 should move down towards lower cab-call")
	assert.Equal(t, behavior, elevator.BMoving, "Elevator 1 should be moving")
}

// CASE 6: Two elevators gets order and you're not supposed to perform it
func TestGetNextOrder_Two_Elevators(t *testing.T) {
	wv, ctx := newTestCtx()

	state := statesync.NewRemoteElevatorState(ID+1, 3, NumFloors)
	wv.AddElevator(state)

	hallCall := statesync.HallCallPair{
		Up: statesync.HallCallPairState{State: statesync.HSAvailable},
	}
	wv.AddHallCall(4, hallCall)

	behavior, dir := ctx.GetNextOrder(ID)
	hallCalls := wv.GetHallCalls()

	require.Equal(t, hallCalls[3].Up.By, ID+1, "Elevator 2 should be assigend to hall-call order")
	assert.Equal(t, dir, elevio.MDStop, "Expected Elevator 1 to stay still")
	assert.Equal(t, behavior, elevator.BIdle, "Elevator 1 should be idle")
}
