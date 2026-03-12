package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	"github.com/Mosazghi/elevator-ttk4145/internal/fsm"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/network"
	"github.com/Mosazghi/elevator-ttk4145/internal/orders"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/lmittmann/tint"
)

func main() {
	cfg := config.Parse()

	SetupLogger(cfg.LogLevel)

	// Check environment mode (ENV=production or ENV=prod enables production mode)
	slog.Info("Elevator started with", "id", cfg.Id, "port", cfg.Port, "floors", cfg.Floors)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	eIOAddr := fmt.Sprintf("localhost:%d", cfg.Port)
	elevIoDriver := elevio.NewElevIoDriver(eIOAddr, cfg.Floors)

	go elevIoDriver.PollButtons(drvButtons)
	go elevIoDriver.PollFloorSensor(drvFloors)
	go elevIoDriver.PollObstructionSwitch(drvObstr)
	go elevIoDriver.PollStopButton(drvStop)

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver)

	network, err := network.NewNetwork()
	if err != nil {
		slog.Error("failed to create network", "err", err)
		return
	}

	defer network.Close()

	network.Start()

	errChan := network.ErrChan()
	txChan := network.TxChan()
	rxChan := network.RxChan()

	triggerAction := make(chan controller.ControllerTriggerSrc, 3*cfg.Floors)
	orderUpdateChan := make(chan statesync.Order, 10)
	actionChan := make(chan any, 10)
	wvChan := make(chan statesync.Worldview, 20)
	wv := statesync.NewWorldView(cfg.Id, cfg.Floors, wvChan, orderUpdateChan)

	ctrller := controller.NewController(wv, actionChan, triggerAction, orderUpdateChan)
	go ctrller.Start()
	go wv.StartSyncing(txChan, rxChan, errChan)

	orderHandler := orders.NewOrderHandler(wvChan, triggerAction, actionChan)
	go orderHandler.Run()

	localElvevator := wv.GetRemoteElevator()
	initFloor := elevIoDriver.GetFloor()
	if initFloor == -1 {
		elev.OnInitBetweenFloors()
		localElvevator.Behavior = elevator.BMoving
		localElvevator.Direction = elevio.MDDown

	}

	localElvevator.CurrentFloor = 0

	err = wv.SetLocalElevator(&localElvevator)
	if err != nil {
		slog.Error("SetLocalElevator", "error", err)
	}

	// Sync all button lights with the current worldview state before entering the
	// main event loop, so the elevator server matches any persisted/recovered calls.
	{
		localElev := wv.GetRemoteElevator()
		hallCallStates := wv.GetAllHallCalls()
		hallCallBools := make([][2]bool, cfg.Floors)
		for floor, pair := range hallCallStates {
			hallCallBools[floor][0] = pair[statesync.HDDown].State != statesync.HSNone
			hallCallBools[floor][1] = pair[statesync.HDUp].State != statesync.HSNone
		}
		elev.SetAllLights(cfg.Floors, localElev.CabCalls, hallCallBools)
	}

	fsm := fsm.NewStateMachine(
		drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		triggerAction,
		actionChan,
		&elev,
		wv,
	)

	fsm.Run()
}

func SetupLogger(level slog.Leveler) {
	w := os.Stderr
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.DateTime,
			AddSource:  true,
		}),
	))
}
