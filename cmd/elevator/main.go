package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	"github.com/Mosazghi/elevator-ttk4145/internal/network"
	"github.com/Mosazghi/elevator-ttk4145/internal/orchestrator"
	"github.com/Mosazghi/elevator-ttk4145/internal/order_handler"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/lmittmann/tint"
)

func main() {
	cfg := config.Parse()

	SetupLogger(cfg.LogLevel)

	slog.Info("Elevator started with", "id", cfg.Id, "port", cfg.Port, "floors", cfg.Floors, "env", os.Getenv("ENV"))

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

	elevatorService := elevator.NewElevatorService(elevIoDriver)

	network, err := network.NewNetwork()
	if err != nil {
		slog.Error("failed to create network", "err", err)
		return
	}
	defer network.Close()

	network.Start()

	errChan := network.GetErrorChannel()
	txChan := network.GetTransmitChannel()
	rxChan := network.GetReceiveChannel()

	triggerAction := make(chan controller.ControllerTriggerSrc, 3*cfg.Floors)
	orderUpdateChan := make(chan statesync.Order, 10)
	actionChan := make(chan any, 10)
	recoveredCabCallChan := make(chan Empty, 5)

	wv := statesync.NewWorldView(cfg.Id, cfg.Floors, orderUpdateChan, recoveredCabCallChan)
	controller := controller.NewController(wv, actionChan, triggerAction, orderUpdateChan)
	orderHandler := order_handler.NewOrderHandler(wv, triggerAction, actionChan)

	go wv.StartSyncing(txChan, rxChan, errChan)
	go controller.StartHandlingRequests()
	go orderHandler.Run()

	localElvevator := wv.GetRemoteElevatorStates()
	initFloor := elevIoDriver.GetFloor()
	if initFloor == -1 {
		elevatorService.SetMoveDirection(elevio.MDDown)
		localElvevator.Direction = elevio.MDDown
		localElvevator.Behavior = elevator.BMoving
	}

	elevatorService.ClearAllLights(localElvevator.NumFloors)

	err = wv.SetLocalElevatorStates(&localElvevator)
	if err != nil {
		slog.Error("SetLocalElevator", "error", err)
	}

	// Sync all button lights with the current worldview state before entering the
	// main event loop, so the elevator server matches any persisted/recovered calls.
	{
		time.Sleep(1500 * time.Millisecond)
		elevatorService.SetDoorState(false)
		hallCallStates := wv.GetAllHallCalls()
		hallCalls := make([][2]bool, wv.NumFloors)
		cabCalls := localElvevator.CabCalls
		for floor, pair := range hallCallStates {
			hallCalls[floor][0] = pair[statesync.HDDown].State != statesync.HallCallStateNone
			hallCalls[floor][1] = pair[statesync.HDUp].State != statesync.HallCallStateNone
		}
		elevatorService.SetAllLights(wv.NumFloors, cabCalls, hallCalls)
	}

	orchestrator := orchestrator.NewOrchestrator(
		drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		triggerAction,
		actionChan,
		recoveredCabCallChan,
		&elevatorService,
		wv,
	)

	orchestrator.Run()
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
