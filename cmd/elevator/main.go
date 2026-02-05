package main

import (
	"flag"
	"fmt"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
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

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver)
	wv := statesync.NewTestWorldview(*id, 4)

	if initFloor == -1 {
		elev.OnInitBetweenFloors()
	}

	stateMachine(drvButtons, drvFloors, drvObstr, drvStop, &elev, wv)
}

func stateMachine(
	drvButtons chan eIO.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	elev *elevator.ElevatorState,
	worldView *statesync.Worldview,
) {
	prevBehavior := elevator.BIdle

	for {

		if prevBehavior != elev.Behavior {
			fmt.Printf("State Trans: %v -> %v\n", prevBehavior, elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {
		case order := <-drvButtons:
			if order.Button == elevio.Cab {
				worldView.SetCabCall(order.Floor, true)
			}

			if order.Button == elevio.HallUp {
				worldView.SetHallCall(order.Floor, statesync.HDUp, statesync.HSAvailable)
			} else {
				worldView.SetHallCall(order.Floor, statesync.HDDown, statesync.HSAvailable)
			}

		// case target := <-orderChan:
		// Do target

		case floor := <-drvFloors:
			worldView.UpdateLocalElevatorFloor(floor)

		case isObstructed := <-drvObst:
			if isObstructed {
				worldView.UpdateLocalElevatorBehavior(elevator.BObstructed)
			}

		case a := <-drvStop:
			elev.OnStopSignal(a)
		}
	}
}
