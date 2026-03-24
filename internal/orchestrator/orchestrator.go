package orchestrator

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"
	"github.com/Mosazghi/elevator-ttk4145/pkg/reinit"
	"github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/Mosazghi/elevator-ttk4145/pkg/watchdog"
)

type Orchestrator struct {
	// Input channels
	drvButtons chan elevio.ButtonEvent
	drvFloors  chan int
	drvObst    chan bool
	drvStop    chan bool
	// Internal channels
	ctrlTriggerChan      chan controller.ControllerTriggerSrc
	ctrlActionChan       chan any
	recoveredCabCallChan chan shared.Empty

	elev *elevator.ElevatorService
	wv   *statesync.Worldview

	watchdog *watchdog.WatchDog
}

func NewOrchestrator(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	ctrlTriggerChan chan controller.ControllerTriggerSrc,
	ctrlActionChan chan any,
	recoveredCabCallChan chan shared.Empty,
	elev *elevator.ElevatorService,
	wv *statesync.Worldview,
) *Orchestrator {
	return &Orchestrator{
		drvButtons:           drvButtons,
		drvFloors:            drvFloors,
		drvObst:              drvObst,
		drvStop:              drvStop,
		ctrlTriggerChan:      ctrlTriggerChan,
		ctrlActionChan:       ctrlActionChan,
		recoveredCabCallChan: recoveredCabCallChan,
		elev:                 elev,
		wv:                   wv,
		watchdog:             watchdog.New(WatchdogTimeout),
	}
}

func (sm *Orchestrator) Run() {
	go sm.watchdog.Start()
	defer sm.watchdog.Stop()

	watchdogTicker := time.NewTicker(WatchdogInterval)
	defer watchdogTicker.Stop()

	for {
		localElvevator := sm.wv.GetRemoteElevatorStates()

		select {
		case <-watchdogTicker.C:
			sm.watchdog.Ping()
		case <-sm.watchdog.Timeout:
			slog.Error("Watchdog timedout.. restarting")
			reinit.Reinitialize()
		case order := <-sm.drvButtons:
			if localElvevator.UndefinedState() {
				continue
			}
			err := sm.makeNewOrder(order)
			if err != nil {
				slog.Error("Failed to make new order", "error", err)
				continue
			}

			if order.Button == elevio.Cab {
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate
			}

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevatorStates(&localElvevator)
			if err != nil {
				slog.Error("SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			sm.ctrlTriggerChan <- controller.CTSFArrivalFloor

		case <-sm.recoveredCabCallChan:
			localElev := sm.wv.GetRemoteElevatorStates()
			sm.elev.SetCabCallLights(sm.wv.NumFloors, localElev.CabCalls)

			sm.ctrlTriggerChan <- controller.CTSOrderUpdate

		case action := <-sm.ctrlActionChan:
			switch action := action.(type) {
			case elevator.MoveAction:
				err := sm.elev.SetMoveDirection(action.Direction)
				if err != nil {
					slog.Error("failed to set action", "err", err)
				}

				localElvevator.Behavior = action.Behavior
				localElvevator.Direction = action.Direction

				sm.wv.SetLocalElevatorStates(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}

			case elevator.StopAction:
				sm.elev.StopMotor()

				localElvevator.Behavior = action.Behavior
				err := sm.wv.SetLocalElevatorStates(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate

			case elevator.LightAction:
				sm.elev.SetCallLight(action.ButtonType, action.Floor, action.State)

			case elevator.DoorAction:
				if !action.Open {
					sm.ctrlTriggerChan <- controller.CTSOrderUpdate
				}
				sm.elev.SetDoorState(action.Open)
			default:
				slog.Warn("Received unknown action type in state machine", "type", fmt.Sprintf("%T", action))
			}

		case isObstructed := <-sm.drvObst:
			localElvevator.IsObstructed = isObstructed

			err := sm.wv.SetLocalElevatorStates(&localElvevator)
			if err != nil {
				slog.Error("SetLocalElevator", "error", err)
			}

			if isObstructed && localElvevator.Behavior == elevator.BDoorOpen {
				sm.elev.StopMotor()
			} else {
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate
			}

		case shouldStop := <-sm.drvStop:
			if shouldStop {
				sm.elev.StopMotor()
				sm.elev.SetStopLight(elevator.LightOn)
			} else {
				sm.elev.SetStopLight(elevator.LightOff)
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate
			}

		}
	}
}

func (sm *Orchestrator) makeNewOrder(order elevio.ButtonEvent) error {
	var err error
	switch order.Button {
	case elevio.Cab:
		err = sm.wv.SetCabCall(order.Floor, true)
	case elevio.HallUp:
		err = sm.wv.NewHallCall(order.Floor, statesync.HallCallDirectionUp)
	case elevio.HallDown:
		err = sm.wv.NewHallCall(order.Floor, statesync.HallCallDirectionDown)
	}

	return err
}
