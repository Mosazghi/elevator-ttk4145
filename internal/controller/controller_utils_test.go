package controller

import (
	"testing"

	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/stretchr/testify/assert"
)

const utilsTestID = 1
const utilsNumFloors = 4

func newElev(id, floor int, dir elevio.MotorDirection) statesync.RemoteElevatorState {
	e := statesync.NewRemoteElevatorState(id, utilsNumFloors)
	e.CurrentFloor = floor
	e.Direction = dir
	return *e
}

func emptyCalls() Calls {
	return Calls{
		HallCalls: make([][2]statesync.HallCallPairState, utilsNumFloors),
		CabCalls:  make([]bool, utilsNumFloors),
	}
}

func hallProcessing(calls *Calls, floor int, dir statesync.HallCallDir, byID int) {
	calls.HallCalls[floor][dir] = statesync.HallCallPairState{
		State: statesync.HSProcessing,
		By:    byID,
	}
}

// ── HasOrdersAbove ────────────────────────────────────────────────────────────

func TestHasOrdersAbove_NoOrders(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDUp)
	calls := emptyCalls()
	assert.False(t, HasOrdersAbove(e, calls))
}

func TestHasOrdersAbove_HallCallAbove(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDUp)
	calls := emptyCalls()
	hallProcessing(&calls, 3, statesync.HDUp, utilsTestID)
	assert.True(t, HasOrdersAbove(e, calls))
}

func TestHasOrdersAbove_HallCallBelowIgnored(t *testing.T) {
	e := newElev(utilsTestID, 2, elevio.MDUp)
	calls := emptyCalls()
	hallProcessing(&calls, 0, statesync.HDDown, utilsTestID)
	assert.False(t, HasOrdersAbove(e, calls))
}

func TestHasOrdersAbove_CabCallAbove(t *testing.T) {
	e := newElev(utilsTestID, 0, elevio.MDUp)
	e.CabCalls[3] = true // HasOrdersAbove checks e.CabCalls, not calls.CabCalls
	calls := emptyCalls()
	assert.True(t, HasOrdersAbove(e, calls))
}

func TestHasOrdersAbove_HallCallOtherElevatorIgnored(t *testing.T) {
	e := newElev(utilsTestID, 0, elevio.MDUp)
	calls := emptyCalls()
	hallProcessing(&calls, 3, statesync.HDUp, 99) // different elevator ID
	calls.CabCalls[3] = false
	assert.False(t, HasOrdersAbove(e, calls))
}

// ── HasOrdersBelow ────────────────────────────────────────────────────────────

func TestHasOrdersBelow_NoOrders(t *testing.T) {
	e := newElev(utilsTestID, 2, elevio.MDDown)
	calls := emptyCalls()
	assert.False(t, HasOrdersBelow(e, calls))
}

func TestHasOrdersBelow_HallCallBelow(t *testing.T) {
	e := newElev(utilsTestID, 3, elevio.MDDown)
	calls := emptyCalls()
	hallProcessing(&calls, 0, statesync.HDDown, utilsTestID)
	assert.True(t, HasOrdersBelow(e, calls))
}

func TestHasOrdersBelow_HallCallAboveIgnored(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDDown)
	calls := emptyCalls()
	hallProcessing(&calls, 3, statesync.HDUp, utilsTestID)
	assert.False(t, HasOrdersBelow(e, calls))
}

func TestHasOrdersBelow_CabCallBelow(t *testing.T) {
	e := newElev(utilsTestID, 3, elevio.MDDown)
	calls := emptyCalls()
	calls.CabCalls[0] = true
	assert.True(t, HasOrdersBelow(e, calls))
}

// ── ShouldStop ────────────────────────────────────────────────────────────────

func TestShouldStop_MDStop_AlwaysTrue(t *testing.T) {
	e := newElev(utilsTestID, 2, elevio.MDStop)
	calls := emptyCalls()
	assert.True(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingUp_HallCallAtFloor(t *testing.T) {
	e := newElev(utilsTestID, 2, elevio.MDUp)
	calls := emptyCalls()
	hallProcessing(&calls, 2, statesync.HDUp, utilsTestID)
	assert.True(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingUp_NoOrdersAbove(t *testing.T) {
	e := newElev(utilsTestID, 3, elevio.MDUp) // top floor, nothing above
	calls := emptyCalls()
	assert.True(t, ShouldStop(e, calls)) // no orders above => should stop
}

func TestShouldStop_MovingUp_OrdersAbove_NoCallAtFloor(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDUp)
	calls := emptyCalls()
	hallProcessing(&calls, 3, statesync.HDUp, utilsTestID) // order above, not at current floor
	assert.False(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingDown_HallCallAtFloor(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDDown)
	calls := emptyCalls()
	hallProcessing(&calls, 1, statesync.HDDown, utilsTestID)
	assert.True(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingDown_CabCallAtFloor(t *testing.T) {
	e := newElev(utilsTestID, 1, elevio.MDDown)
	calls := emptyCalls()
	calls.CabCalls[1] = true
	assert.True(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingDown_NoOrdersBelow(t *testing.T) {
	e := newElev(utilsTestID, 0, elevio.MDDown) // ground floor, nothing below
	calls := emptyCalls()
	assert.True(t, ShouldStop(e, calls))
}

func TestShouldStop_MovingDown_OrdersBelow_NoCallAtFloor(t *testing.T) {
	e := newElev(utilsTestID, 3, elevio.MDDown)
	calls := emptyCalls()
	hallProcessing(&calls, 0, statesync.HDDown, utilsTestID) // order below, not at current floor
	assert.False(t, ShouldStop(e, calls))
}
