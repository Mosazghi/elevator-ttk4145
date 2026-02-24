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
				// if(requests_shouldClearImmediately(*e, btn_floor, btn_type)){
				//     timer_start(e->config.doorOpenDuration_s);
				// } else {
				//     e->requests[btn_floor][btn_type] = 1;
				// }
				// break;
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
				// DirnBehaviourPair pair = requests_chooseDirection(*e);
				// e->dirn = pair.dirn;
				// e->behaviour = pair.behaviour;
				// switch(pair.behaviour){
				// case EB_DoorOpen:
				//     elevator_doorLight(1);
				//     timer_start(e->config.doorOpenDuration_s);
				//     *e = requests_clearAtCurrentFloor(*e);
				//     break;

				// case EB_Moving:
				//     elevator_motorDirection(e->dirn);
				//     break;

				// case EB_Idle:
				//     break;
				// }
				// break;
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
							sm.elev.SetCallLight(btnType, floor, elevator.LSOn)
						} else {

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
