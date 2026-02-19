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
		slog.Info("Running in production mode (echo filtering enabled)")
	} else {
		slog.Info("Running in development mode (echo filtering disabled)")
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

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver, elevIoDriver.GetFloor())

	network, err := network.NewNetwork(prodMode)
	if err != nil {
		slog.Error("failed to create network", "err", err)
		return
	}

	defer network.Close()

	network.Start()

	txChan := network.TxChan()
	rxChan := network.RxChan()
	errChan := network.ErrChan()

	wvChan := make(chan statesync.Worldview, 20)
	actionChan := make(chan elevator.Action)
	wv := statesync.NewWorldView(*localID, 4, wvChan)

	go wv.StartSyncing(txChan, rxChan, errChan)
	go orders.GetNextAction(wvChan, actionChan)

	localElvevator := wv.GetRemoteElevator()
	localElvevator.CurrentFloor = elevIoDriver.GetFloor()

	err = wv.SetLocalElevator(&localElvevator)
	if err != nil {
		slog.Error("[StateMachine] SetLocalElevator", "error", err)
	}

	stateMachine(drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		actionChan,
		&elev,
		wv)
}

// /FIXME: Not receivng new worldview when cab call
func stateMachine(
	drvButtons chan elevio.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	actionChan chan elevator.Action,
	elev *elevator.ElevatorState,
	wv *statesync.Worldview,
) {
	prevBehavior := elevator.BIdle

	for {
		localElvevator := wv.GetRemoteElevator()

		if prevBehavior != elev.Behavior {
			slog.Info("[StateMachine] Transition", "prevBehavior", prevBehavior, "current Behavior", elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {
		case a := <-drvButtons:
			slog.Debug("Button event received", "event", a)
			var err error
			switch a.Button {
			case elevio.Cab:
				err = wv.SetCabCall(a.Floor, true)
			case elevio.HallUp:
				err = wv.NewHallCall(a.Floor, statesync.HDUp)
			case elevio.HallDown:
			default:
				err = wv.NewHallCall(a.Floor, statesync.HDDown)
			}

			if err != nil {
				slog.Error("failed to set new cab/hall call", "err", err)
			}

		// case order := <-drvButtons:
		// 	var hcDir statesync.HallCallDir
		//
		// 	if order.Button == elevio.HallUp {
		// 		hcDir = statesync.HDUp
		// 	} else {
		// 		hcDir = statesync.HDDown
		// 	}
		//
		// 	if order.Button == elevio.Cab {
		// 		slog.Info("[StateMachine] Set cabcall", "floor", order.Floor)
		// 		err := wv.SetCabCall(order.Floor, true)
		// 		if err != nil {
		// 			slog.Error("[StateMachine] SetCabCall", "err", err)
		// 		}
		// 		elev.SetCallLight(elevio.Cab, order.Floor, elevator.LSOn)
		// 	} else {
		// 		err := wv.NewHallCall(order.Floor, hcDir)
		// 		if err != nil {
		// 			slog.Error("[StateMachine] SetHallCall", "error", err)
		// 		}
		// 	}

		case floor := <-drvFloors:
			localElvevator.CurrentFloor = floor
			err := wv.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetLocalElevator", "error", err)
			}
			elev.SetCurrentFloorLight(floor)

		case action := <-actionChan:
			err := elev.SetAction(action)
			if err != nil {
				slog.Error("[StateMachine] SetAction", "Behavior", action.Behavior.String(), "Direction", action.Direction.String())
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

func SetupLogger() {
	w := os.Stderr
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.DateTime,
		}),
	))
}
