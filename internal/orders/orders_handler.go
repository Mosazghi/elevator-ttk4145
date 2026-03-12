package orders

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os/exec"

	"github.com/Mosazghi/elevator-ttk4145/internal/controller"
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	"github.com/Mosazghi/elevator-ttk4145/shared"
)

type ElevatorCost struct {
	id    int
	floor int
	cost  int
}

type OrderHandler struct {
	wvChan    chan statesync.Worldview
	trigger   chan controller.ControllerTriggerSrc
	actionCah chan any
}

func NewOrderHandler(wvChan chan statesync.Worldview, trigger chan controller.ControllerTriggerSrc, actionChan chan any) *OrderHandler {
	return &OrderHandler{
		wvChan:    wvChan,
		trigger:   trigger,
		actionCah: actionChan,
	}
}

func (o *OrderHandler) Run() {
	for wv := range o.wvChan {
		hallCalls := wv.GetAllHallCalls()

		for floor, hallCall := range hallCalls {
			for dir := range hallCalls[floor] {
				if hallCall[dir].State == statesync.HSNone {
					continue
				}

				// count how many that are not alive

				aliveCount := 0
				for _, elev := range wv.ElevatorStates {
					if elev.Alive {
						aliveCount++
					}
				}

				isConfirmedByAll := len(hallCall[dir].ConfirmedBy) >= aliveCount
				isAvailable := hallCall[dir].State == statesync.HSAvailable

				if !isConfirmedByAll || !isAvailable {
					continue
				}

				winner := CalculateCost(&wv, floor, statesync.HallCallDir(dir))
				if winner.id == wv.LocalID {
					err := wv.ProcessHallCall(floor, statesync.HallCallDir(dir))
					if err != nil {
						slog.Error("[RunCost] Got worldview error", "error", err)
					}
					slog.Info("[RunCost] Set to processing", "floor", floor, "Direction", dir, "id", winner.id)

					o.trigger <- controller.CTSOrderUpdate
				}
				slog.Warn("Order picked up", "floor", floor, "dir", dir, "by", winner.id)
			}
		}
	}
}

type hallRequestAssignerElevState struct {
	Behavior    string `json:"behavior"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HallRequestAssignerInput struct {
	HallRequests [][2]bool                               `json:"hallRequests"`
	States       map[string]hallRequestAssignerElevState `json:"states"`
}

func behaviorToString(b elevator.Behavior) string {
	switch b {
	case elevator.BIdle:
		return "idle"
	case elevator.BMoving:
		return "moving"
	case elevator.BDoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}

func directionToString(d elevio.MotorDirection) string {
	switch d {
	case elevio.MDUp:
		return "up"
	case elevio.MDDown:
		return "down"
	default:
		return "stop"
	}
}

type HallRequestAssignerOutput map[string][][2]bool

func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}
	hrs := BuildHallRequestAssignerData(wv)

	// execute command to the hall call assinger at ../simulat/hall_request_assigner
	slog.Debug("HallRequestAssigner data", "data", hrs)

	// -i | --input : JSON input.
	//     Example: ./hall_request_assigner --input '{"hallRequests":....}'
	// --travelDuration : Travel time between two floors in milliseconds (default 2500)
	// --doorOpenDuration : Door open time in milliseconds (default 3000)
	// --clearRequestType : When stopping at a floor, clear either all requests or only those inDirn (default)
	// --includeCab : Includes the cab requests in the output. The output becomes a 3xN boolean matrix for each elevator ([[up-0, down-0, cab-0], [...],...]). (disabled by default)

	jsonData, err := json.Marshal(hrs)
	if err != nil {
		slog.Error("[CalculateCost] Failed to marshal hall request assigner data", "error", err)
		return winner
	}

	execCmd := exec.Command("simulator/hall_request_assigner", "--input", string(jsonData))
	output, err := execCmd.Output()
	if err != nil {
		slog.Error("[CalculateCost] Failed to execute hall request assigner", "error", err)
		return winner
	}

	var result HallRequestAssignerOutput

	err = json.Unmarshal(output, &result)
	// the winner is who got true in the corresponding floor and direction
	if err != nil {
		slog.Error("[CalculateCost] Failed to unmarshal hall request assigner output", "error", err)
		return winner
	}

	slog.Debug("HallRequestAssigner output", "output", result)

	for id, assigned := range result {
		if assigned[floor][dir] {
			// convert id to int
			var intID int
			_, err := fmt.Sscanf(id, "%d", &intID)
			if err != nil {
				slog.Error("[CalculateCost] Failed to parse elevator ID from hall request assigner output", "error", err, "id", id)
				continue
			}
			winner.id = intID
			winner.floor = floor
			break
		}
	}

	return winner
}

func BuildHallRequestAssignerData(wv *statesync.Worldview) *HallRequestAssignerInput {
	hcs := wv.GetAllHallCalls()
	hrs := HallRequestAssignerInput{
		HallRequests: make([][2]bool, wv.NumFloors),
		States:       make(map[string]hallRequestAssignerElevState),
	}

	// Build hall calls
	for floor, pair := range hcs {
		for dir, state := range pair {
			if state.State == statesync.HSAvailable {
				hrs.HallRequests[floor][dir] = true
			}
		}
	}

	// Build elevator states
	for id, elev := range wv.ElevatorStates {
		if !elev.Alive {
			continue
		}

		hrs.States[fmt.Sprintf("%d", id)] = hallRequestAssignerElevState{
			Behavior:    behaviorToString(elev.Behavior),
			Floor:       elev.CurrentFloor,
			Direction:   directionToString(elev.Direction),
			CabRequests: elev.CabCalls,
		}
	}

	return &hrs
}

func CalculateCost_(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}

	for id, elev := range wv.ElevatorStates {
		if !elev.Alive {
			continue
		}

		currentElevatorCost := ElevatorCost{-1, wv.NumFloors + 1, 0}
		isObstructed := elev.IsObstructed

		currentElevatorCost.id = id

		if !elev.Alive {
			continue
		}

		if isObstructed {
			currentElevatorCost.cost += shared.PenaltyObstructed
		}

		// If the elevator is moving in the wrong direction, add a penalty to the cost
		if elev.Direction == elevio.MDDown && dir == statesync.HDUp {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDUp && dir == statesync.HDDown {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		// If the elevator is moving away from the call, add a penalty to the cost
		if elev.Direction == elevio.MDUp && elev.CurrentFloor >= floor {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		if elev.Direction == elevio.MDDown && elev.CurrentFloor <= floor {
			currentElevatorCost.cost += shared.PenaltyWrongDirection
		}

		distance := int(math.Abs(float64(elev.CurrentFloor - floor)))

		currentElevatorCost.cost += distance

		slog.Debug("[CalculateCost] Cost for elevator", "id", id, "cost", currentElevatorCost.cost, "distance", distance, "isObstructed", isObstructed, "currentDir", elev.Direction)

		if currentElevatorCost.cost < winner.cost || (currentElevatorCost.cost == winner.cost && currentElevatorCost.id < winner.id) {
			winner.id = currentElevatorCost.id
			winner.cost = currentElevatorCost.cost
			winner.floor = currentElevatorCost.floor
		}
	}
	return winner
}
