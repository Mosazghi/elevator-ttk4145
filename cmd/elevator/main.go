package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	eIO "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
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

	// initFloor := elevIoDriver.GetFloor()

	elev := elevator.NewElevState(initFloor, elevIoDriver.ReadInitialButtons(), elevIoDriver)

	// Start network
	txChan, rxChan, errChan, err := network.Start(prodMode)
	if err != nil {
		slog.Error("Failed to start network", "error", err)
		return
	}
	wvChan := make(chan statesync.Worldview, 10)
	wv := statesync.NewWorldView(*localID, 4, wvChan)
	go wv.StartSyncing(txChan, rxChan)
	stateMachine(drvButtons, drvFloors, drvObstr, drvStop, elev, wv)
}

func stateMachine(
	drvButtons chan eIO.ButtonEvent,
	drvFloors chan int,
	drvObst chan bool,
	drvStop chan bool,
	elev *elevator.ElevatorState,
	worldView *statesync.Worldview,
) {
	local_elv := worldView.GetLocalElevator()

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
		case msg := <-rxChan:
			fmt.Printf("Received: %s from %s\n", string(msg.Data), msg.Address.String())

		case err := <-errChan:
			fmt.Printf("Network error: %v\n", err)

		case <-ticker.C:
			txChan <- network.UDPMessage{Data: []byte("Hello from A")}
			fmt.Println("Sent broadcast message")
		case order := <-drvButtons:
			fmt.Println("Got new order!")
			if order.Button == elevio.Cab {
				fmt.Println("Cab call!")
				if order.Floor > local_elv.CurrentFloor {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDUp})
				} else {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDDown})
				}

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
				}
				// worldView.SetCabCall(order.Floor, true)
			}

			if order.Button == elevio.HallUp {
				fmt.Println("Hall call up")
				elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDUp})

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
				}

				// worldView.SetHallCall(order.Floor, statesync.HDUp, statesync.HSAvailable)
			}

			if order.Button == elevio.HallDown {
				fmt.Println("Hall call down")
				elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDDown})

				if order.Floor == local_elv.CurrentFloor {
					fmt.Println("Allready on floor")
				}
				// worldView.SetHallCall(order.Floor, statesync.HDDown, statesync.HSAvailable)
			}

		case floor := <-drvFloors:
			worldView.UpdateLocalElevatorFloor(floor)

		case isObstructed := <-drvObst:
			if isObstructed {
				// worldView.UpdateLocalElevatorBehavior(elevator.BObstructed)
				elev.Stop()
			} else {
				elev.Continue()
			}

		case shouldStop := <-drvStop:
			if shouldStop {
				elev.Stop()
			} else {
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
