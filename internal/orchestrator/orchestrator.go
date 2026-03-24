// Package orchestrator coordinates elevator input, local state, controller decisions,
// and hardware actions for a single elevator node in the distributed system.
//
// Orchestrator reads driver events (buttons, floor sensors, obstruction, stop),
// maintains a local watchdog, updates the shared world view, and forwards
// control triggers and action commands between the controller and elevator hardware.
//
// It also handles recovery of pending cab calls from state sync and restarts on watchdog timeout.
package orchestrator

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	"github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	"github.com/Mosazghi/elevator-ttk4145/pkg/reinit"
	"github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/Mosazghi/elevator-ttk4145/pkg/watchdog"
)

// An Orchestrator maintains the flow of the program. It takes the input channels and internal channels
// to both regiser and complete orders.
//
// It has an internal watchdog timer to ensure that the system is working properly.
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
	elevatorService      *elevator.ElevatorService
	worldview            *statesync.Worldview
	watchdog             *watchdog.WatchDog
}

// NewOrchestrator creates a new instance of an initialized Orchestrator
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
		elevatorService:      elev,
		worldview:            wv,
		watchdog:             watchdog.New(WatchdogTimeout),
	}
}

// Start is the main loop for orchestrator operation.
//
// It processes:
//   - driver input events coming from hw channels,
//   - controller action results,
//   - recovered cab call notifications,
//   - watchdog pings/timeouts.
//
// On watchdog timeout, it reintializes the entire system.
func (orchestrator *Orchestrator) Start() {
	go orchestrator.watchdog.Start()
	defer orchestrator.watchdog.Stop()

	watchdogTicker := time.NewTicker(WatchdogInterval)
	defer watchdogTicker.Stop()

	for {

		select {
		case <-watchdogTicker.C:
			orchestrator.watchdog.Ping()
		case <-orchestrator.watchdog.TimeoutChan:
			slog.Error("Watchdog timedout.. restarting")
			reinit.Reinitialize()
		case order := <-orchestrator.drvButtons:
			localElvevator := orchestrator.worldview.GetRemoteElevatorState()
			if localElvevator.UndefinedState() {
				continue
			}
			err := orchestrator.makeNewOrder(order)
			if err != nil {
				slog.Error("Failed to make new order", "error", err)
				continue
			}

			if order.Button == elevio.Cab {
				orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate
			}

		case floor := <-orchestrator.drvFloors:
			localElvevator := orchestrator.worldview.GetRemoteElevatorState()
			localElvevator.CurrentFloor = floor
			err := orchestrator.worldview.SetLocalElevatorStates(&localElvevator)
			if err != nil {
				slog.Error("SetLocalElevator", "error", err)
			}
			orchestrator.elevatorService.SetCurrentFloorLight(floor)
			orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcArrivalFloor

		case <-orchestrator.recoveredCabCallChan:
			localElev := orchestrator.worldview.GetRemoteElevatorState()
			orchestrator.elevatorService.SetCabCallLights(orchestrator.worldview.NumFloors, localElev.CabCalls)

			orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate

		case action := <-orchestrator.ctrlActionChan:
			switch action := action.(type) {
			case elevator.MoveAction:
				err := orchestrator.elevatorService.SetMoveDirection(action.Direction)
				if err != nil {
					slog.Error("failed to set action", "err", err)
				}

				localElvevator := orchestrator.worldview.GetRemoteElevatorState()
				localElvevator.Behavior = action.Behavior
				localElvevator.Direction = action.Direction

				orchestrator.worldview.SetLocalElevatorStates(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}

			case elevator.StopAction:
				orchestrator.elevatorService.StopMotor()

				localElvevator := orchestrator.worldview.GetRemoteElevatorState()
				localElvevator.Behavior = action.Behavior
				err := orchestrator.worldview.SetLocalElevatorStates(&localElvevator)
				if err != nil {
					slog.Error("SetLocalElevator", "err", err)
				}
				orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate

			case elevator.LightAction:
				orchestrator.elevatorService.SetCallLight(action.ButtonType, action.Floor, action.State)

			case elevator.DoorAction:
				if !action.Open {
					orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate
				}
				orchestrator.elevatorService.SetDoorState(action.Open)
			default:
				slog.Warn("Received unknown action type in state machine", "type", fmt.Sprintf("%T", action))
			}

		case isObstructed := <-orchestrator.drvObst:
			localElvevator := orchestrator.worldview.GetRemoteElevatorState()
			localElvevator.IsObstructed = isObstructed

			err := orchestrator.worldview.SetLocalElevatorStates(&localElvevator)
			if err != nil {
				slog.Error("SetLocalElevator", "error", err)
			}

			if isObstructed && localElvevator.Behavior == elevator.BehaviorDoorOpen {
				orchestrator.elevatorService.StopMotor()
			} else {
				orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate
			}

		case shouldStop := <-orchestrator.drvStop:
			if shouldStop {
				orchestrator.elevatorService.StopMotor()
				orchestrator.elevatorService.SetStopLight(true)
			} else {
				orchestrator.elevatorService.SetStopLight(false)
				orchestrator.ctrlTriggerChan <- controller.ControllerTriggerSrcOrderUpdate
			}

		}
	}
}

// makeNewOrder tries to creates a new order and updates the world view
func (orchestrator *Orchestrator) makeNewOrder(order elevio.ButtonEvent) error {
	var err error
	switch order.Button {
	case elevio.Cab:
		err = orchestrator.worldview.SetCabCall(order.Floor, true)
	case elevio.HallUp:
		err = orchestrator.worldview.NewHallCall(order.Floor, statesync.HallCallDirectionUp)
	case elevio.HallDown:
		err = orchestrator.worldview.NewHallCall(order.Floor, statesync.HallCallDirectionDown)
	}

	return err
}
