package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	elevator "github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/lmittmann/tint"
	// network "github.com/Mosazghi/elevator-ttk4145/internal/net"
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
	slog.Info("Elevator started", "id", *localID)
	slog.Info("Elevator started", "port", *port)

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

	elev := elevator.NewElevator(elevator.BIdle, elevio.MDStop, elevIoDriver)

	wvChan := make(chan statesync.Worldview, 10)
	wv := statesync.NewWorldView(1, 4, wvChan)

	localElvevator := wv.GetRemoteElevator()
	localElvevator.CurrentFloor = elevIoDriver.GetFloor()
	err := wv.SetLocalElevator(&localElvevator)
	if err != nil {
		slog.Error("[StateMachine] SetHallCall", "error", err)
	}

	stateMachine(drvButtons,
		drvFloors,
		drvObstr,
		drvStop,
		&elev,
		wv)
}

func stateMachine(
	drvButtons chan elevio.ButtonEvent,
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
	hallCallDir := 0
	goal := 0

	for {
		localElvevator := worldView.GetRemoteElevator()

		if prevBehavior != elev.Behavior {
			slog.Info("[StateMachine] Transition", "prevBehavior", prevBehavior, "current Behavior", elev.Behavior)
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
			if order.Floor > localElvevator.CurrentFloor {
				hallCallDir = statesync.HDUp
			} else {
				hallCallDir = statesync.HDDown
			}

			goal = order.Floor
			elev.SetDoor(elevator.DSClosed)

			if order.Button == elevio.Cab {
				elev.SetCallLight(elevio.Cab, order.Floor, elevator.LSOn)

				if order.Floor == localElvevator.CurrentFloor {
					slog.Info("[StateMachine] Allready on floor")
					continue
				}

				if order.Floor > localElvevator.CurrentFloor {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDUp})
				} else {
					elev.SetAction(elevator.Action{elevator.BMoving, elevio.MDDown})
				}

				// worldView.SetCabCall(order.Floor, true)
			} else {
				err := worldView.SetHallCall(order.Floor, statesync.HallCallDir(hallCallDir), statesync.HSAvailable)
				if err != nil {
					slog.Error("[StateMachine] SetHallCall", "error", err)
				}

			}

		case floor := <-drvFloors:
			localElvevator.CurrentFloor = floor
			err := worldView.SetLocalElevator(&localElvevator)
			if err != nil {
				slog.Error("[StateMachine] SetHallCall", "error", err)
			}

			elev.SetCurrentFloorLight(floor)
			slog.Info("[StateMachine] Reached new floor", "floor", floor, "goal", goal)

			if goal == floor {
				elev.SetAction(elevator.Action{elevator.BIdle, elevio.MDStop})
				elev.SetCallLight(elevio.Cab, goal, elevator.LSOff)
				elev.SetDoor(elevator.DSOpen)
			}

		// Our understanding: Cannot accur a obstruction during movment
		// Example: someone is infront of the door!
		// Obstruct means we cannot close the door
		// Obsructuion is only resolved/accur during open door not movement
		case isObstructed := <-drvObst:
			if isObstructed {
				// worldView.UpdateLocalElevatorBehavior(elevator.BObstructed)
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
			TimeFormat: time.Kitchen,
		}),
	))
}
