package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	//"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	//eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"

	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
	"github.com/lmittmann/tint"
)

// var numFloors = 4
func main() {

	w := os.Stderr

	// Create a new logger

	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
	port := flag.Int("port", 15657, "specify port number")
	localID := flag.Int("id", 1, "specify elevator ID")

	flag.Parse()
	slog.Info("Elevator started", "id", *localID)
	slog.Info("Elevator started", "port", *port)

	// 	drvButtons := make(chan eIO.ButtonEvent)
	// 	drvFloors := make(chan int)
	// 	drvObstr := make(chan bool)
	// 	drvStop := make(chan bool)

	// 	elevIoDriver := eIO.NewElevIoDriver("localhost:"+*portNum, 4)

	// 	go elevIoDriver.PollButtons(drvButtons)
	// 	go elevIoDriver.PollFloorSensor(drvFloors)
	// 	go elevIoDriver.PollObstructionSwitch(drvObstr)
	// 	go elevIoDriver.PollStopButton(drvStop)

	// 	initFloor := elevIoDriver.GetFloor()

	// 	elev := elevator.NewElevState(initFloor, elevIoDriver.ReadInitialButtons(), elevIoDriver)

	// 	if initFloor == -1 {
	// 		elev.OnInitBetweenFloors()
	// 	}

	// Start network
	txChan, rxChan, errChan, err := network.UDPRunNetwork()
	if err != nil {
		fmt.Printf("Failed to start network: %v\n", err)
		return
	}
	wv := statesync.NewWorldView(*localID, 4)
	wv.StartSyncing(txChan, rxChan, errChan)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Handle all channels
	// for {
	// 	select {
	// 	case msg := <-rxChan:
	// 		fmt.Printf("Received: %s from %s\n", string(msg.Data), msg.Address)

	// 	case err := <-errChan:
	// 		fmt.Printf("Network error: %v\n", err)

	// 	case <-ticker.C:
	// 		msg := fmt.Sprintf("Hello from %d", *localID)
	// 		txChan <- network.UDPMessage{Data: []byte(msg)}
	// 		fmt.Println("Sent broadcast message")
	// 	}
	// }

	// stateMachine(drvButtons, drvFloors, drvObstr, drvStop, elev)
}

// func stateMachine(drvButtons chan eIO.ButtonEvent, drvFloors chan int, drvObst chan bool, drvStop chan bool, elev *elevator.ElevState) {

// 	prevBehavior := elevator.BIdle

// 	for {

// 		if prevBehavior != elev.Behavior {
// 			fmt.Printf("State Trans: %v -> %v\n", prevBehavior, elev.Behavior)
// 			prevBehavior = elev.Behavior
// 		}

// 		select {
// 		case a := <-drvButtons:
// 			elev.OnOrderRequest(a)
// 		case a := <-drvFloors:
// 			elev.OnNewFloorArrival(a)
// 		case a := <-drvObst:
// 			elev.OnObstructionSignal(a)
// 		case a := <-drvStop:
// 			elev.OnStopSignal(a)
// 		}
// 	}
// }
