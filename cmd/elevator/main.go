package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	//"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	//eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/lmittmann/tint"
)

// var numFloors = 4
func main() {
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
	slog.Info("Elevator started", "id", *localID)
	slog.Info("Elevator started", "port", *port)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	eIOAddr := fmt.Sprintf("localhost:%d", *port)
	elevIoDriver := elevio.NewElevIoDriver(eIOAddr, 4)

	go elevIoDriver.PollButtons(drvButtons)

	go elevIoDriver.PollFloorSensor(drvFloors)
	go elevIoDriver.PollObstructionSwitch(drvObstr)
	go elevIoDriver.PollStopButton(drvStop)

	initFloor := elevIoDriver.GetFloor()

	elev := elevator.NewElevState(initFloor, elevIoDriver.ReadInitialButtons(), elevIoDriver)

	if initFloor == -1 {
		slog.Info("Elevator initialized between floors")
		elev.OnInitBetweenFloors()
	}

	// Start network
	txChan, rxChan, errChan, err := network.Start(prodMode)
	if err != nil {
		slog.Error("Failed to start network", "error", err)
		return
	}
	wvChan := make(chan statesync.Worldview, 10)
	wv := statesync.NewWorldView(*localID, 4, wvChan)

	go wv.StartSyncing(txChan, rxChan, errChan)

	stateMachine(drvButtons, drvFloors, drvObstr, drvStop, elev, wv)
}

func stateMachine(drvButtons chan elevio.ButtonEvent, drvFloors chan int, drvObst chan bool, drvStop chan bool, elev *elevator.ElevState, wv *statesync.Worldview) {
	prevBehavior := elevator.BIdle

	for {

		if prevBehavior != elev.Behavior {
			slog.Info("State Trans", "from", prevBehavior, "to", elev.Behavior)
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
				err = wv.SetHallCall(a.Floor, statesync.HDUp, statesync.HSAvailable)
			default:
				err = wv.SetHallCall(a.Floor, statesync.HDDown, statesync.HSAvailable)
			}
			if err != nil {
				slog.Error("Failed to set call in worldview", "error", err)
			}

			elev.OnOrderRequest(a)
		case a := <-drvFloors:
			elev.OnNewFloorArrival(a)
		case a := <-drvObst:
			elev.OnObstructionSignal(a)
		case a := <-drvStop:
			elev.OnStopSignal(a)
		}
	}
}

func SetupLogger() {
	w := os.Stderr
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
}
