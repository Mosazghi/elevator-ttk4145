package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/internal/net"
)

var numFloors = 4

func main() {
	portNum := flag.String("port", "15657", "specify port number")
	id := flag.Int("id", 1, "specify elevator ID")

	fmt.Println("ID: ", *id)

	flag.Parse()

	drvButtons := make(chan eIO.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	elevIoDriver := eIO.NewElevIoDriver("localhost:"+*portNum, 4)

	go elevIoDriver.PollButtons(drvButtons)
	go elevIoDriver.PollFloorSensor(drvFloors)
	go elevIoDriver.PollObstructionSwitch(drvObstr)
	go elevIoDriver.PollStopButton(drvStop)

	initFloor := elevIoDriver.GetFloor()

	elev := elevator.NewElevState(initFloor, elevIoDriver.ReadInitialButtons(), elevIoDriver)

	if initFloor == -1 {
		elev.OnInitBetweenFloors()
	}

	// Start network
    txChan, rxChan, errChan, err := network.UDPRunNetwork("nodeA")
    if err != nil {
        fmt.Printf("Failed to start network: %v\n", err)
        return
    }

	ticker := time.NewTicker(2* time.Second)
	defer ticker.Stop()
    // Handle all channels
    for {
        select {
        case msg := <-rxChan:
            fmt.Printf("Received: %s from %s\n", string(msg.Data), msg.Address.String())
            
        case err := <-errChan:
            fmt.Printf("Network error: %v\n", err)
            
        case <-ticker.C:
            // Send a message
            txChan <- network.UDPMessage{Data: []byte("Hello from A")}
        }
    }

	stateMachine(drvButtons, drvFloors, drvObstr, drvStop, elev)
}

func stateMachine(drvButtons chan eIO.ButtonEvent, drvFloors chan int, drvObst chan bool, drvStop chan bool, elev *elevator.ElevState) {

	prevBehavior := elevator.BIdle

	for {

		if prevBehavior != elev.Behavior {
			fmt.Printf("State Trans: %v -> %v\n", prevBehavior, elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {
		case a := <-drvButtons:
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
