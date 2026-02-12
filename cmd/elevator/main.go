package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	elevator "github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/lmittmann/tint"

	// network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
)

var numFloors = 4

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

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver)
	wv := statesync.NewWorldView(1, 4)

	initFloor := elevIoDriver.GetFloor()
	if initFloor == -1 {
		wv.UpdateLocalElevatorFloor(0)
	}

	stateMachine(drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		&elev,
		wv)
}

func stateMachine(
	drvButtons chan eIO.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	elev *elevator.ElevatorState,
	worldView *statesync.Worldview,
) {
	// // Start network
	// txChan, rxChan, errChan, err := network.UDPRunNetwork()
	// if err != nil {
	// 	fmt.Printf("Failed to start network: %v\n", err)
	// 	return
	// }
	//
	// ticker := time.NewTicker(2 * time.Second)
	// defer ticker.Stop()
	prevBehavior := elevator.BIdle
	goal := 0
	stopped := false

	if prevBehavior != elev.Behavior {
		fmt.Printf("State Trans: %v -> %v\n", prevBehavior, elev.Behavior)
		prevBehavior = elev.Behavior
	}

	for {

		local_elv := worldView.GetLocalElevator()

		if prevBehavior != elev.Behavior {
			fmt.Printf("State Trans: %v -> %v\n", prevBehavior, elev.Behavior)
			prevBehavior = elev.Behavior
		}

		select {
		// case msg := <-rxChan:
		// 	fmt.Printf("Received: %s from %s\n", string(msg.Data), msg.Address.String())
		//
		// case err := <-errChan:
		// 	fmt.Printf("Network error: %v\n", err)
		//
		// case <-ticker.C:
		// 	txChan <- network.UDPMessage{Data: []byte("Hello from A")}
		// 	fmt.Println("Sent broadcast message")

		case order := <-drvButtons:
			goal = order.Floor
			elev.SetDoor(elevator.DSClosed)
			fmt.Println("Goal: ", goal)
			if order.Button == elevio.Cab {
				elev.SetCallLight(elevio.Cab, order.Floor, elevator.LSOn)
				fmt.Println("Cab call!")

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
					continue
				}

				if order.Floor > local_elv.CurrentFloor {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDUp})
				} else {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDDown})
				}

				// worldView.SetCabCall(order.Floor, true)
			}

			if order.Button == elevio.HallUp {
				fmt.Println("Hall call up")

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
					continue
				}
				elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDUp})
				// worldView.SetHallCall(order.Floor, statesync.HDUp, statesync.HSAvailable)
			}

			if order.Button == elevio.HallDown {
				fmt.Println("Hall call down")

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
					continue
				}
				elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDDown})
				// worldView.SetHallCall(order.Floor, statesync.HDDown, statesync.HSAvailable)
			}

		case floor := <-drvFloors:
			worldView.UpdateLocalElevatorFloor(floor)
			elev.SetCurrentFloorLight(floor)
			fmt.Println("floor: ", floor, " goal: ", goal)

			if goal == floor {
				elev.SetAction(elevator.Action{elevator.BIdle, elevio.MDStop})
				elev.SetCallLight(elevio.Cab, goal, elevator.LSOff)
				elev.SetDoor(elevator.DSOpen)
			}

		// Our understanding: Cannot accur a obstruction during movment
		// Example: someone is infront of the door!
		// Obstruct means we cannot close the door
		case isObstructed := <-drvObst:
			if isObstructed {
				// worldView.UpdateLocalElevatorBehavior(elevator.BObstructed)
				elev.Stop()
			} else {
				elev.Continue()
			}

		case shouldStop := <-drvStop:
			if shouldStop {
				stopped = true
				elev.Stop()
				elev.SetStopLight(elevator.LSOn)
				if stopped {
					stopped = false
				}
			}

			if !stopped {
				elev.SetStopLight(elevator.LSOff)
				elev.Continue()
			}
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
