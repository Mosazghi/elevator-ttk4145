package fsm

import (
	"fmt"
	"log/slog"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type StateMachine struct {
	drvButtons    chan elevio.ButtonEvent
	drvFloors     chan int
	drvObst       chan bool
	drvStop       chan bool
	arriveAtFloor chan struct{}
	actionChan    chan any
	anyOrder      chan struct{}
	elev          *elevator.ElevatorState
	wv            *statesync.Worldview
}

func NewStateMachine(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	arriveAtFloor chan struct{},
	actionChan chan any,
	anyOrder chan struct{},
	elev *elevator.ElevatorState,
	wv *statesync.Worldview,
) *StateMachine {
	return &StateMachine{
		drvButtons:    drvButtons,
		drvFloors:     drvFloors,
		drvObst:       drvObst,
		drvStop:       drvStop,
		arriveAtFloor: arriveAtFloor,
		actionChan:    actionChan,
		anyOrder:      anyOrder,
		elev:          elev,
		wv:            wv,
	}
}

func (sm *StateMachine) Run() {
	prevBehavior := elevator.BIdle

	for {
		localElvevator := sm.wv.GetRemoteElevator()

		if prevBehavior != sm.elev.Behavior {
			prevBehavior = sm.elev.Behavior
		}

		// if controller.ShouldClearImmediately(localElvevator, order.Floor, order.Button) {
		// 	slog.Error("Should clear immediately", "floor", order.Floor, "button", order.Button)
		// } else {
		// 	err := sm.makeNewOrder(order)
		// 	if err != nil {
		// 		slog.Error("Failed to make new order", "error", err)
		// 	}
		// }
		select {
		case order := <-sm.drvButtons:
			err := sm.makeNewOrder(order)
			if err != nil {
				slog.Error("Failed to make new order", "error", err)
			}

			if order.Button == elevio.Cab {
				sm.anyOrder <- struct{}{}
			}

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			slog.Debug("[arriveAtFloor] trigger")
			sm.arriveAtFloor <- struct{}{}

		case action := <-sm.actionChan:
			slog.Debug("[StateMachine] Received action", "type", fmt.Sprintf("%T", action), "value", action)
			switch action := action.(type) {
			case elevator.MoveAction:
				err := sm.elev.DoMotorAction(action)
				if err != nil {
					slog.Error("failed to set action", "err", err)
				}

				localElvevator.Behavior = action.Behavior
				localElvevator.Direction = action.Direction
				sm.wv.SetLocalElevator(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}

			case elevator.LightAction:
				sm.elev.SetCallLight(action.ButtonType, action.Floor, action.State)
			case elevator.DoorAction:
				if !action.Open {
					slog.Debug("[anyOrder] trigger (at door close)")
					sm.anyOrder <- struct{}{}
				}
				sm.elev.SetDoor(action.Open)
				// sm.elev.SetCallLight(elevio.Cab, localElvevator.CurrentFloor, false)

			default:
				slog.Warn("Received unknown action type in state machine", "type", fmt.Sprintf("%T", action))
			}

		// FIXME: Implement logic for this
		// Our understanding: Cannot accur a obstruction during movment
		// Example: someone is infront of the door!
		// Obstruct means we cannot close the door
		// Obsructuion is only resolved/accur during open door not movement
		case isObstructed := <-sm.drvObst:
			localElvevator.IsObstructed = isObstructed

			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}

		case shouldStop := <-sm.drvStop:
			if shouldStop {
				sm.elev.StopAction()
				sm.elev.SetStopLight(elevator.LSOn)
			} else {
				sm.elev.SetStopLight(elevator.LSOff)
				sm.elev.ContinueAction()
			}

		}
	}
}

func (sm *StateMachine) makeNewOrder(order elevio.ButtonEvent) error {
	var err error
	switch order.Button {
	case elevio.Cab:
		err = sm.wv.SetCabCall(order.Floor, true)
		sm.elev.SetCallLight(order.Button, order.Floor, true)
	case elevio.HallUp:
		err = sm.wv.NewHallCall(order.Floor, statesync.HDUp)
	case elevio.HallDown:
		err = sm.wv.NewHallCall(order.Floor, statesync.HDDown)
	}

	if err != nil {
		slog.Error("failed to set new cab/hall call", "err", err)
	}
	return err
}
