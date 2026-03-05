package fsm

import (
	"log/slog"

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
			slog.Info("[StateMachine] Transition", "prevBehavior", prevBehavior, "current Behavior", sm.elev.Behavior)
			prevBehavior = sm.elev.Behavior
		}

		select {
		case order := <-sm.drvButtons:
			slog.Debug("Button event received", "event", order)
			var err error
			switch order.Button {
			case elevio.Cab:
				err = sm.wv.SetCabCall(order.Floor, true)
				select {
				case sm.trigger <- struct{}{}:
				default:
				}
			case elevio.HallUp:
				err = sm.wv.NewHallCall(order.Floor, statesync.HDUp)
			case elevio.HallDown:
				err = sm.wv.NewHallCall(order.Floor, statesync.HDDown)
			}

			if err != nil {
				slog.Error("failed to set new cab/hall call", "err", err)
			}

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			select {
			case sm.trigger <- struct{}{}:
			default:
			}

		case action := <-sm.actionChan:
			switch a := action.(type) {
			case elevator.MoveAction:
				err := sm.elev.DoMotorAction(a)
				slog.Info("[StateMachine] SetAction", "Behavior", a.Behavior.String(), "Direction", a.Direction.String())
				if err != nil {
					slog.Error("[StateMachine] SetAction", "Behavior", a.Behavior.String(), "Direction", a.Direction.String())
				}
			case elevator.StopAction:
				slog.Info("[StateMachine] StopAction received, stopping elevator")
				sm.elev.DoMotorAction(elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop})
				// select {
				// case sm.trigger <- struct{}{}:
				// default:
				// }

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
		sm.elev.SetCallLight(order.Button, order.Floor, elevator.LSOn)
		select {
		case sm.trigger <- struct{}{}:
		default:
		}
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
