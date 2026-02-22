package fsm

import (
	"fmt"
	"log/slog"
	"time"

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

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			controller.OnFloorArrival(sm.wv, sm.actionChan)

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

			case elevator.SingleLightAction:
				sm.elev.SetCallLight(action.ButtonType, action.Floor, action.State)
			case elevator.SetAllLightsAction:
				hcs := sm.wv.GetAllHallCalls()
				slog.Info("Hc", "hc", hcs)
				for floor, active := range localElvevator.CabCalls {
					if active {
						sm.elev.SetCallLight(elevio.Cab, floor, elevator.LSOn)
					} else {
						sm.elev.SetCallLight(elevio.Cab, floor, elevator.LSOff)
					}
				}

				for floor := range hcs {
					for dir, state := range hcs[floor] {
						btnType := elevio.HallUp
						if dir == int(statesync.HDDown) {
							btnType = elevio.HallDown
						}

						if state.State != statesync.HSNone {
							slog.Warn("Setting light for hall call",
								"floor", floor,
								"direction", dir,
								"state", state.State)
							sm.elev.SetCallLight(btnType, floor, elevator.LSOn)
						} else {

							slog.Warn("Clearing light for hall call",
								"floor", floor,
								"direction", dir,
								"state", state.State)
							sm.elev.SetCallLight(btnType, floor, elevator.LSOff)
						}
					}
				}
			case elevator.DoorAction:
				if action.Open {
					sm.elev.SetDoor(elevator.DSOpen)
				} else {
					sm.elev.SetDoor(elevator.DSClosed)
				}
			default:
				slog.Warn("Received unknown action type in state machine", "type", fmt.Sprintf("%T", action))

				// sm.elev.SetCallLight()
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

// DO NOT DELETE: Used for simulation and testing
func stateMachineSimulationOnly(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	elev *elevator.ElevatorState,
	wv *statesync.Worldview,
) {
	prevBehavior := elevator.BIdle
	goal := 0

	for {
		localElvevator := wv.GetRemoteElevator()

		if prevBehavior != elev.Behavior {
			slog.Info("[StateMachine] Transition", "prevBehavior", prevBehavior, "current Behavior", elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {

		case order := <-drvButtons:
			var err error
			var hc statesync.HallCallDir

			switch order.Button {
			case elevio.Cab:
				err = wv.SetCabCall(order.Floor, true)
			case elevio.HallUp:
				err = wv.NewHallCall(order.Floor, statesync.HDUp)
				hc = statesync.HDUp
			case elevio.HallDown:
				err = wv.NewHallCall(order.Floor, statesync.HDDown)
				hc = statesync.HDDown
			}

			if err != nil {
				slog.Error("failed to set new cab/hall call", "err", err)
			}

			slog.Warn("sleeping...")

			time.Sleep(1000 * time.Millisecond)
			err = wv.ProcessHallCall(order.Floor, hc)
			slog.Info("should have processed")

			if err != nil {
				slog.Error("failed to process hall call", "err", err)
			}

		case floor := <-drvFloors:
			localElvevator.CurrentFloor = floor
			err := wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}

			elev.SetCurrentFloorLight(floor)
			slog.Info("[StateMachine] Reached new floor", "floor", floor, "goal", goal)

			if goal == floor {
				elev.DoMotorAction(elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop})
				elev.SetCallLight(elevio.Cab, goal, elevator.LSOff)
				elev.SetDoor(elevator.DSOpen)
			}

			err = wv.CompleteHallCall(floor, statesync.HDDown)
			if err != nil {
				slog.Error("[StateMachine] CompleteHallCall", "error", err)
			}
			err = wv.CompleteHallCall(floor, statesync.HDUp)
			if err != nil {
				slog.Error("[StateMachine] CompleteHallCall", "error", err)
			}

		// FIXME: Implement logic for this
		// Our understanding: Cannot accur a obstruction during movment
		// Example: someone is infront of the door!
		// Obstruct means we cannot close the door
		// Obsructuion is only resolved/accur during open door not movement
		case isObstructed := <-drvObst:
			localElvevator.IsObstructed = isObstructed
			wv.SetLocalElevator(&localElvevator)

		case shouldStop := <-drvStop:
			if shouldStop {
				elev.StopAction()
				elev.SetStopLight(elevator.LSOn)
			} else {
				elev.SetStopLight(elevator.LSOff)
				elev.ContinueAction()

			}
			elev.DoMotorAction(elevator.MoveAction{Behavior: elevator.BIdle, Direction: elevio.MDStop})
		}
	}
}
