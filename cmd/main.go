package main

import (
	"flag"
	"fmt"

	. "Heisern/pkg/elevator"
	"Heisern/pkg/elevio"
)

var numFloors = 4

func main() {
	portNum := flag.String("port", "15657", "specify port number")
	flag.Parse()

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	elevIoDriver := elevio.NewElevIoDriver("localhost:"+*portNum, 4)

	go elevIoDriver.PollButtons(drvButtons)
	go elevIoDriver.PollFloorSensor(drvFloors)
	go elevIoDriver.PollObstructionSwitch(drvObstr)
	go elevIoDriver.PollStopButton(drvStop)

	stateMachine(drvButtons, drvFloors, drvObstr, drvStop, elevIoDriver)
}

func stateMachine(drvButtons chan elevio.ButtonEvent, drvFloors chan int, drvObst chan bool, drvStop chan bool, elevIoDriver *elevio.ElevIoDriver) {
	initFloor := elevIoDriver.GetFloor()

	elev := NewElevState(initFloor, elevIoDriver.ReadInitialButtons(), elevIoDriver)

	if initFloor == -1 {
		elev.OnInitBetweenFloors()
	}

	prevBehavior := Idle

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
