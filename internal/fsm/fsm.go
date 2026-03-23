package fsm

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/reinit"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	wd "github.com/Mosazghi/elevator-ttk4145/internal/watchdog"
	"github.com/Mosazghi/elevator-ttk4145/shared"
)

type StateMachine struct {
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

	watchdog *wd.WatchDog
}

func NewStateMachine(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	ctrlTriggerChan chan controller.ControllerTriggerSrc,
	ctrlActionChan chan any,
	recoveredCabCallChan chan shared.Empty,
	elev *elevator.ElevatorService,
	wv *statesync.Worldview,
) *StateMachine {
	return &StateMachine{
		drvButtons:           drvButtons,
		drvFloors:            drvFloors,
		drvObst:              drvObst,
		drvStop:              drvStop,
		ctrlTriggerChan:      ctrlTriggerChan,
		ctrlActionChan:       ctrlActionChan,
		recoveredCabCallChan: recoveredCabCallChan,
		elev:                 elev,
		wv:                   wv,
		watchdog:             wd.New(5 * time.Second),
	}
}

func (sm *StateMachine) Run() {
	go sm.watchdog.Start()
	defer sm.watchdog.Stop()

	watchdogTicker := time.NewTicker(1 * time.Second)
	defer watchdogTicker.Stop()

	for {
		localElvevator := sm.wv.GetRemoteElevator()

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
			// ISSUE: If a order is process in a valid state, then it WILL NOT change direction (move until stop logic)
			// Solution: I need to stop after reaching a floor for the internal start, and only then

			err := sm.makeNewOrder(order)
			if err != nil {
				slog.Error("Failed to make new order", "error", err)
			}

			if order.Button == elevio.Cab {
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate
			}

		case floor := <-sm.drvFloors:
			localElvevator.CurrentFloor = floor
			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			sm.elev.SetCurrentFloorLight(floor)
			sm.ctrlTriggerChan <- controller.CTSFArrivalFloor

		case <-sm.recoveredCabCallChan:
			localElev := sm.wv.GetRemoteElevator()
			sm.elev.SetCabCallLights(sm.wv.NumFloors, localElev.CabCalls)

			sm.ctrlTriggerChan <- controller.CTSOrderUpdate

		case action := <-sm.ctrlActionChan:
			// slog.Debug("[StateMachine] Received action", "type", fmt.Sprintf("%T", action), "value", action)
			switch action := action.(type) {
			case elevator.MoveAction:
				err := sm.elev.MoveDirection(action.Direction)
				if err != nil {
					slog.Error("failed to set action", "err", err)
				}

				localElvevator.Behavior = action.Behavior
				localElvevator.Direction = action.Direction

				sm.wv.SetLocalElevator(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}

			case elevator.StopAction:
				sm.elev.Stop()

				localElvevator.Behavior = action.Behavior
				err := sm.wv.SetLocalElevator(&localElvevator)
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
				sm.elev.SetDoor(action.Open)
			default:
				slog.Warn("Received unknown action type in state machine", "type", fmt.Sprintf("%T", action))
			}

		case isObstructed := <-sm.drvObst:
			localElvevator.IsObstructed = isObstructed

			err := sm.wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}

			if isObstructed && localElvevator.Behavior == elevator.BDoorOpen {
				sm.elev.Stop()
			} else {
				sm.ctrlTriggerChan <- controller.CTSOrderUpdate
			}

		case shouldStop := <-sm.drvStop:
			if shouldStop {
				sm.elev.Stop()
				sm.elev.SetStopLight(elevator.LightOn)
			} else {
				sm.elev.SetStopLight(elevator.LightOff)
				sm.elev.MoveDirection(localElvevator.Direction)
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
