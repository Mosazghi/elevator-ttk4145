package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	"github.com/Mosazghi/elevator-ttk4145/internal/orders"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/lmittmann/tint"
)

func main() {
	numFloors := 4
	port := flag.Int("port", 15657, "specify port number")
	localID := flag.Int("id", 1, "specify elevator ID")

	SetupLogger()

	flag.Parse()

	// Check environment mode (ENV=production or ENV=prod enables production mode)
	env := os.Getenv("ENV")
	prodMode := env == "production" || env == "prod"
	if prodMode {
		slog.Warn("Running in production mode (echo filtering enabled)")
	} else {
		slog.Warn("Running in development mode (echo filtering disabled)")
	}
	slog.Info("Elevator started with", "id", *localID, "port", *port)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	eIOAddr := fmt.Sprintf("localhost:%d", *port)
	elevIoDriver := elevio.NewElevIoDriver(eIOAddr, numFloors)

	go elevIoDriver.PollButtons(drvButtons)
	go elevIoDriver.PollFloorSensor(drvFloors)
	go elevIoDriver.PollObstructionSwitch(drvObstr)
	go elevIoDriver.PollStopButton(drvStop)

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver)

	network, err := network.NewNetwork(prodMode)
	if err != nil {
		slog.Error("failed to create network", "err", err)
		return
	}

	defer network.Close()

	network.Start()

	errChan := network.ErrChan()
	txChan := network.TxChan()
	rxChan := network.RxChan()

	triggerAction := make(chan struct{}, 1)
	floorTrigger := make(chan statesync.Worldview, 1)
	actionChan := make(chan any)
	wvChan := make(chan statesync.Worldview, 20)
	wv := statesync.NewWorldView(*localID, 4, wvChan)

	go orders.FSM(floorTrigger, actionChan)
	go orders.GetNextAction(wv, triggerAction, actionChan)
	go wv.StartSyncing(txChan, rxChan, errChan)
	go orders.RunCost(wvChan, triggerAction, actionChan)

	initFloor := elevIoDriver.GetFloor()
	if initFloor == -1 {
		slog.Warn("Elevator is between floors, moving down to the nearest floor")
		elev.OnInitBetweenFloors()

	}

	localElvevator := wv.GetRemoteElevator()
	localElvevator.CurrentFloor = 0

	err = wv.SetLocalElevator(&localElvevator)
	if err != nil {
		slog.Error("[StateMachine] SetLocalElevator", "error", err)
	}

	stateMachine(drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		triggerAction,
		floorTrigger,
		actionChan,
		&elev,
		*wv)
}

func stateMachine(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	trigger chan struct{},
	floorTrigger chan statesync.Worldview,
	actionChan chan any,
	elev *elevator.ElevatorState,
	wv statesync.Worldview,
) {
	prevBehavior := elevator.BIdle

	for {
		localElvevator := wv.GetRemoteElevator()

		if prevBehavior != elev.Behavior {
			slog.Info("[StateMachine] Transition", "prevBehavior", prevBehavior, "current Behavior", elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {
		case order := <-drvButtons:
			slog.Debug("Button event received", "event", order)
			var err error
			switch order.Button {
			case elevio.Cab:
				err = wv.SetCabCall(order.Floor, true)
				select {
				case trigger <- struct{}{}:
				default:
				}
			case elevio.HallUp:
				err = wv.NewHallCall(order.Floor, statesync.HDUp)
			case elevio.HallDown:
				err = wv.NewHallCall(order.Floor, statesync.HDDown)
			}

			if err != nil {
				slog.Error("failed to set new cab/hall call", "err", err)
			}

		case floor := <-drvFloors:
			localElvevator.CurrentFloor = floor
			err := wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			elev.SetCurrentFloorLight(floor)
			select {
			case floorTrigger <- wv:
			default:
			}

		case action := <-actionChan:
			slog.Debug("Action received", "action", fmt.Sprintf("%T", action), "value", action)
			switch action := action.(type) {
			case elevator.MoveAction:
				err := elev.DoMotorAction(action)
				slog.Info("[StateMachine] SetAction", "Behavior", action.Behavior.String(), "Direction", action.Direction.String())

				if err != nil {
					slog.Error("failed to set action", "err", err)
				}

				if action.Behavior == elevator.BDoorOpen {
					remote := wv.GetRemoteElevator()
					rawHallCalls := wv.GetAllHallCalls()

					hallLights := make([][2]bool, remote.NumFloors)
					for f, pair := range rawHallCalls {
						hallLights[f][statesync.HDDown] = pair[statesync.HDDown].State != statesync.HSNone
						hallLights[f][statesync.HDUp] = pair[statesync.HDUp].State != statesync.HSNone
					}

					elev.SetAllLights(remote.NumFloors, remote.CabCalls, hallLights)
					// elev.SetCallLight()
				}
			case elevator.DoorAction:
				if action.Open {
					elev.SetDoor(elevator.DSOpen)
				} else {
					elev.SetDoor(elevator.DSClosed)
				}
			case elevator.LightAction:
				elev.SetCallLight(action.ButtonType, action.Floor, action.State)
			}

		// FIXME: Implement logic for this
		// Our understanding: Cannot accur a obstruction during movment
		// Example: someone is infront of the door!
		// Obstruct means we cannot close the door
		// Obsructuion is only resolved/accur during open door not movement
		case isObstructed := <-drvObst:
			if isObstructed {
				localElvevator.Behavior = elevator.BObstructed
			} else {
				localElvevator.Behavior = elevator.BIdle
			}

			err := wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}

		case shouldStop := <-drvStop:
			if shouldStop {
				elev.StopAction()
				elev.SetStopLight(elevator.LSOn)
			} else {
				elev.SetStopLight(elevator.LSOff)
				elev.ContinueAction()
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

func SetupLogger() {
	w := os.Stderr
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.DateTime,
		}),
	))
}
