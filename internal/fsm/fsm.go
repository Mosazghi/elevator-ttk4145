package fsm

import (
	"fmt"
	"log/slog"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type StateMachine struct {
	drvButtons chan elevio.ButtonEvent
	drvFloors  chan int
	drvObst    chan bool
	drvStop    chan bool
	trigger    chan struct{}
	actionChan chan any
	elev       *elevator.ElevatorState
	wv         *statesync.Worldview
}

func NewStateMachine(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	trigger chan struct{},
	actionChan chan any,
	elev *elevator.ElevatorState,
	wv *statesync.Worldview,
) *StateMachine {
	return &StateMachine{
		drvButtons: drvButtons,
		drvFloors:  drvFloors,
		drvObst:    drvObst,
		drvStop:    drvStop,
		trigger:    trigger,
		actionChan: actionChan,
		elev:       elev,
		wv:         wv,
	}
}

func (sm *StateMachine) Run() {
	prevBehavior := elevator.BIdle

	for {
		localElvevator := sm.wv.GetRemoteElevator()

		if prevBehavior != sm.elev.Behavior {
			prevBehavior = sm.elev.Behavior
		}

		select {
		case order := <-sm.drvButtons:
			switch localElvevator.Behavior {
			case elevator.BDoorOpen:
				if controller.ShouldClearImmediately(localElvevator, order.Floor, order.Button) {
					slog.Info("Should clear immediately", "floor", order.Floor, "button", order.Button)
				} else {
					err := sm.makeNewOrder(order)
					if err != nil {
						slog.Error("Failed to make new order", "error", err)
					}
				}
			case elevator.BMoving:
				err := sm.makeNewOrder(order)
				if err != nil {
					slog.Error("Failed to make new order", "error", err)
				}
			case elevator.BIdle:
				err := sm.makeNewOrder(order)
				if err != nil {
					slog.Error("Failed to make new order", "error", err)
				}
			}

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			slog.Info("[StateMachine] Trigger after floor arrival", "floor", floor)
			sm.trigger <- struct{}{}

		case action := <-sm.actionChan:
			slog.Info("[StateMachine] Received action", "type", fmt.Sprintf("%T", action), "value", action)
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
					slog.Info("Trigger after door close")
					sm.trigger <- struct{}{}
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
			if isObstructed {
				localElvevator.Behavior = elevator.BObstructed
			} else {
				localElvevator.Behavior = elevator.BIdle
			}

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
		sm.trigger <- struct{}{}
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
