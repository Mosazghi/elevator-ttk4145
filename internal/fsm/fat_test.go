package fsm

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/orders"
	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

// FakeDriver implements elevio.ElevatorDriver and stores lamp state in memory
type FakeDriver struct {
	mu         sync.Mutex
	HallLights [][2]bool             // [floor][up/down]
	CabLights  []bool
	DoorOpen   bool
	StopLight  bool
	FloorLight int
	MotorDir   elevio.MotorDirection
}

func NewFakeDriver(floors int) *FakeDriver {
	return &FakeDriver{
		HallLights: make([][2]bool, floors),
		CabLights:  make([]bool, floors),
		FloorLight: 0,
		MotorDir:   elevio.MDStop,
	}
}

func (d *FakeDriver) SetButtonLamp(button elevio.ButtonType, floor int, value bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if button == elevio.HallUp {
		d.HallLights[floor][0] = value
	} else if button == elevio.HallDown {
		d.HallLights[floor][1] = value
	} else if button == elevio.Cab {
		d.CabLights[floor] = value
	}
}

func (d *FakeDriver) SetDoorOpenLamp(value bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.DoorOpen = value
}

func (d *FakeDriver) SetStopLamp(value bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.StopLight = value
}

func (d *FakeDriver) SetFloorIndicator(floor int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.FloorLight = floor
}

func (d *FakeDriver) SetMotorDirection(dir elevio.MotorDirection) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.MotorDir = dir
}

// Read helpers for assertions
func (d *FakeDriver) HallLight(floor int, btn elevio.ButtonType) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if btn == elevio.HallUp {
		return d.HallLights[floor][0]
	} else if btn == elevio.HallDown {
		return d.HallLights[floor][1]
	}
	return false
}

func (d *FakeDriver) CabLight(floor int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.CabLights[floor]
}

func (d *FakeDriver) DoorIsOpen() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.DoorOpen
}

func (d *FakeDriver) MotorDirection() elevio.MotorDirection {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.MotorDir
}

func (d *FakeDriver) GetFloor() int {
	return 0
}

func (d *FakeDriver) GetTotalFloors() int {
	return len(d.CabLights)
}

func (d *FakeDriver) ReadInitialButtons() [4][3]bool {
	return [4][3]bool{}
}

func (d *FakeDriver) GetButton(button elevio.ButtonType, floor int) bool {
	return false
}

func (d *FakeDriver) GetStop() bool {
	return false
}

func (d *FakeDriver) GetObstruction() bool {
	return false
}

func (d *FakeDriver) PollButtons(receiver chan<- elevio.ButtonEvent) {}
func (d *FakeDriver) PollFloorSensor(receiver chan<- int)            {}
func (d *FakeDriver) PollStopButton(receiver chan<- bool)            {}
func (d *FakeDriver) PollObstructionSwitch(receiver chan<- bool)     {}

// FakeBus broadcasts messages to all peers
type FakeBus struct {
	mu    sync.Mutex
	peers map[string]chan network.UDPMessage
}

func NewFakeBus() *FakeBus {
	return &FakeBus{
		peers: make(map[string]chan network.UDPMessage),
	}
}

func (b *FakeBus) Register(id string) (tx chan network.UDPMessage, rx chan network.UDPMessage) {
	tx = make(chan network.UDPMessage, 32)
	rx = make(chan network.UDPMessage, 32)

	b.mu.Lock()
	b.peers[id] = rx
	b.mu.Unlock()

	go func() {
		for msg := range tx {
			b.mu.Lock()
			peers := make(map[string]chan network.UDPMessage)
			for peerID, peerRx := range b.peers {
				peers[peerID] = peerRx
			}
			b.mu.Unlock()

			for peerID, peerRx := range peers {
				if peerID == id {
					continue
				}
				select {
				case peerRx <- msg:
				default:
				}
			}
		}
	}()

	return tx, rx
}

// TestNode represents one elevator node with all necessary components
type TestNode struct {
	ID         string
	FSM        *StateMachine
	WV         *statesync.Worldview
	DrvButtons chan elevio.ButtonEvent
	DrvFloors  chan int
	DrvObstr   chan bool
	DrvStop    chan bool
	Trigger    chan struct{}
	ActionChan chan any

	Driver *FakeDriver

	txChan chan network.UDPMessage
	rxChan chan network.UDPMessage
	cancel context.CancelFunc
}

func newTestNode(id string, numID int, floors int, bus *FakeBus) *TestNode {
	drvButtons := make(chan elevio.ButtonEvent, 16)
	drvFloors := make(chan int, 16)
	drvObstr := make(chan bool, 16)
	drvStop := make(chan bool, 16)

	trigger := make(chan struct{}, floors)
	actionChan := make(chan any, 16)
	wvChan := make(chan statesync.Worldview, 20)
	newOrderChan := make(chan statesync.Order, 10)

	driver := NewFakeDriver(floors)
	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, driver)

	wv := statesync.NewWorldView(numID, floors, wvChan, newOrderChan)

	tx, rx := bus.Register(id)

	// Start FSM with context
	ctx, cancel := context.WithCancel(context.Background())

	// Create and start the FSM
	sm := NewStateMachine(
		drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		trigger,
		actionChan,
		&elev,
		wv,
	)
	go sm.RunWithContext(ctx)

	// Start controller
	go ControllerStart(ctx, wv, trigger, actionChan, newOrderChan)

	// Start order handler
	orderHandler := orders.NewOrderHandler(wvChan, trigger, actionChan)
	go OrderHandlerRun(ctx, orderHandler)

	// Start worldview syncing
	errChan := make(chan error, 1)
	go func() {
		wv.StartSyncing(tx, rx, errChan)
	}()

	// Initialize local elevator state
	local := wv.GetRemoteElevator()
	local.CurrentFloor = 0
	_ = wv.SetLocalElevator(&local)

	return &TestNode{
		ID:         id,
		FSM:        sm,
		WV:         wv,
		DrvButtons: drvButtons,
		DrvFloors:  drvFloors,
		DrvObstr:   drvObstr,
		DrvStop:    drvStop,
		Trigger:    trigger,
		ActionChan: actionChan,
		Driver:     driver,
		txChan:     tx,
		rxChan:     rx,
		cancel:     cancel,
	}
}

func (n *TestNode) Stop() {
	n.cancel()
	close(n.DrvButtons)
	close(n.DrvFloors)
	close(n.DrvObstr)
	close(n.DrvStop)
	close(n.Trigger)
	close(n.ActionChan)
	close(n.txChan)
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}

// Helper wrappers for FSM, controller, and order handler to accept context
func StateMachineRun(ctx context.Context, drvButtons chan elevio.ButtonEvent, drvFloors chan int, drvObstr chan bool, drvStop chan bool, trigger chan struct{}, actionChan chan any, elev *elevator.ElevatorState, wv *statesync.Worldview) {
	sm := &StateMachine{
		drvButtons: drvButtons,
		drvFloors:  drvFloors,
		drvObst:    drvObstr,
		drvStop:    drvStop,
		trigger:    trigger,
		actionChan: actionChan,
		elev:       elev,
		wv:         wv,
	}
	sm.RunWithContext(ctx)
}

func ControllerStart(ctx context.Context, wv *statesync.Worldview, trigger chan struct{}, actionChan chan any, newOrderChan chan statesync.Order) {
	for {
		select {
		case <-ctx.Done():
			return
		case order := <-newOrderChan:
			actionChan <- elevator.SingleLightAction{ButtonType: orders.HallDirToButtonType(order.Dir), Floor: order.Floor, State: order.Completed}

		case <-trigger:
			slog.Warn("trigger!!!")
			local := wv.GetRemoteElevator()

			if local.Behavior == elevator.BMoving {
				slog.Warn("Already moving, skipping trigger")
				continue
			}

			nearestHallCall, _ := controller.CalculateNearestHallCall(wv)
			nearestCabCall := controller.CalculateNearestCabCall(wv)

			if nearestHallCall == -1 && nearestCabCall == -1 {
				slog.Info("No pending orders")
				continue
			}

			if nearestCabCall != -1 {
				dur := elevio.MDUp
				diff := nearestCabCall - local.CurrentFloor
				if diff < 0 {
					dur = elevio.MDDown
				}

				elev := wv.GetRemoteElevator()
				if nearestHallCall == -1 {
					actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dur}
				} else {
					hallDistance := nearestHallCall - local.CurrentFloor
					if (diff < 0 && hallDistance < 0 && diff > hallDistance) || (diff > 0 && hallDistance > 0 && diff < hallDistance) || (diff < 0 && hallDistance > 0) {
						actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: dur}
					} else {
						hallDir := elevio.MDUp
						if hallDistance < 0 {
							hallDir = elevio.MDDown
						}
						actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: hallDir}
					}
				}

				elev.Direction = dur
				_ = wv.SetLocalElevator(&elev)
			} else if nearestHallCall != -1 {
				hallDir := elevio.MDUp
				diff := nearestHallCall - local.CurrentFloor
				if diff < 0 {
					hallDir = elevio.MDDown
				}

				actionChan <- elevator.MoveAction{Behavior: elevator.BMoving, Direction: hallDir}

				elev := wv.GetRemoteElevator()
				elev.Direction = hallDir
				_ = wv.SetLocalElevator(&elev)
			}
		}
	}
}

func OrderHandlerRun(ctx context.Context, o *orders.OrderHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case wv := <-o.GetWvChan():
			hallCalls := wv.GetAllHallCalls()

			for floor, hallCall := range hallCalls {
				for dir := range hallCalls[floor] {
					if hallCall[dir].State == statesync.HSNone {
						continue
					}

					isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= len(wv.ElevatorStates)
					isAvailable := hallCall[dir].State == statesync.HSAvailable

					if hallCall[dir].State == statesync.HSProcessing && hallCall[dir].By == wv.LocalID {
						select {
						case o.GetTrigger() <- struct{}{}:
							slog.Info("Triggered from OrderHandler")
						default:
						}
					}

					if !isConfirmedByAll || !isAvailable {
						continue
					}

					winner := orders.CalculateCost(&wv, floor, statesync.HallCallDir(dir))
					if winner.ID == wv.LocalID {
						err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
						if err != nil {
							slog.Error("[RunCost] Got worldview error", "error", err)
						}
						slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir, "id", winner.ID)
					}
					slog.Warn("[RunCost] Order picked up", "by", winner.ID)
					o.GetActionChan() <- elevator.SingleLightAction{ButtonType: orders.HallDirToButtonType(statesync.HallCallDir(dir)), Floor: floor, State: true}
				}
			}
		}
	}
}

// Tests
func TestButtonLightGuarantee(t *testing.T) {
	bus := NewFakeBus()

	n1 := newTestNode("e1", 1, 4, bus)
	defer n1.Stop()

	n2 := newTestNode("e2", 2, 4, bus)
	defer n2.Stop()

	n3 := newTestNode("e3", 3, 4, bus)
	defer n3.Stop()

	// Give nodes time to sync initial state
	time.Sleep(500 * time.Millisecond)

	t.Run("hall call lights sync and one elevator serves it", func(t *testing.T) {
		n1.DrvButtons <- elevio.ButtonEvent{
			Floor:  2,
			Button: elevio.HallUp,
		}

		waitUntil(t, 3*time.Second, func() bool {
			return n1.Driver.HallLight(2, elevio.HallUp) &&
				n2.Driver.HallLight(2, elevio.HallUp) &&
				n3.Driver.HallLight(2, elevio.HallUp)
		})

		// Simulate one elevator serving the order.
		n2.DrvFloors <- 2

		waitUntil(t, 5*time.Second, func() bool {
			return !n1.Driver.HallLight(2, elevio.HallUp) &&
				!n2.Driver.HallLight(2, elevio.HallUp) &&
				!n3.Driver.HallLight(2, elevio.HallUp)
		})
	})

	t.Run("cab call stays local to the originating elevator", func(t *testing.T) {
		n2.DrvButtons <- elevio.ButtonEvent{
			Floor:  3,
			Button: elevio.Cab,
		}

		waitUntil(t, 2*time.Second, func() bool {
			return n2.Driver.CabLight(3)
		})

		if n1.Driver.CabLight(3) {
			t.Fatal("node 1 must not light node 2 cab call")
		}
		if n3.Driver.CabLight(3) {
			t.Fatal("node 3 must not light node 2 cab call")
		}

		n2.DrvFloors <- 3

		waitUntil(t, 2*time.Second, func() bool {
			return !n2.Driver.CabLight(3)
		})
	})
}
