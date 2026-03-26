package order_handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
)

// HallRequestAssignerElevatorState is the per-elevator JSON shape expected by
// the external hall request assigner binary.
type HallRequestAssignerElevatorState struct {
	Behavior    string `json:"behavior"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

// HallRequestAssignerInput is the full JSON payload sent to the assigner.
type HallRequestAssignerInput struct {
	HallRequests [][2]bool                                   `json:"hallRequests"`
	States       map[string]HallRequestAssignerElevatorState `json:"states"`
}

// HallRequestAssignerOutput maps elevator ID to assigned hall calls.
type HallRequestAssignerOutput map[string][][2]bool

// CalculateCost delegates hall-call assignment to the external assigner and
// returns the winning elevator ID for a specific floor/direction.
func CalculateCost(worldview *statesync.Worldview, floor int, direction statesync.HallCallDirection) (int, error) {
	hallRequestAssingerInput := BuildHallRequestAssignerInput(worldview)

	if len(hallRequestAssingerInput.States) == 0 {
		return UnassignedID, fmt.Errorf("no elevators available to assign")
	}

	winnerID := UnassignedID

	inputJson, err := json.Marshal(hallRequestAssingerInput)
	if err != nil {
		return winnerID, fmt.Errorf("failed to marshal hall request assigner data: %w", err)
	}

	hallRequestAssignerPath := getHallReqAssingerPath()

	rawOuput, err := exec.Command(hallRequestAssignerPath, "--input", string(inputJson)).Output()
	if err != nil {
		return winnerID, fmt.Errorf("failed to execute hall request assigner: %w", err)
	}

	var hallRequestAssignerOutput HallRequestAssignerOutput

	err = json.Unmarshal(rawOuput, &hallRequestAssignerOutput)
	if err != nil {
		return winnerID, fmt.Errorf("failed to unmarshal hall request assigner output: %w", err)
	}

	// the winner is who got true in the corresponding floor and direction
	for idString, hallCallAssignments := range hallRequestAssignerOutput {
		if hallCallAssignments[floor][direction] {
			idInt, err := strconv.Atoi(idString)

			if err != nil {
				return winnerID, fmt.Errorf("failed to parse elevator ID from hall request assigner output: %w", err)
			}

			winnerID = idInt
			break
		}
	}

	return winnerID, nil
}

// BuildHallRequestAssignerInput converts worldview state to assigner input,
// excluding elevators that are currently not allowed to serve orders.
func BuildHallRequestAssignerInput(worldview *statesync.Worldview) *HallRequestAssignerInput {
	hallCalls := worldview.GetAllHallCalls()
	hallRequestAssignerInput := HallRequestAssignerInput{
		HallRequests: make([][2]bool, worldview.NumFloors),
		States:       make(map[string]HallRequestAssignerElevatorState),
	}

	// Build hall calls
	for floor, pair := range hallCalls {
		for direction, hallCall := range pair {
			isOrderActive := hallCall.State == statesync.HallCallStateConfirmed || hallCall.State == statesync.HallCallStateProcessing

			if isOrderActive {
				hallRequestAssignerInput.HallRequests[floor][direction] = true
			}
		}
	}

	// Build elevator states
	for id, elevator := range worldview.ElevatorStates {
		if !elevator.IsAllowedToServe() {
			continue
		}

		hallRequestAssignerInput.States[strconv.Itoa(id)] = HallRequestAssignerElevatorState{
			Behavior:    elevator.Behavior.String(),
			Floor:       elevator.CurrentFloor,
			Direction:   elevator.Direction.String(),
			CabRequests: elevator.CabCalls,
		}
	}

	return &hallRequestAssignerInput
}

// getHallReqAssingerPath resolves the assigner binary path, preferring a path
// relative to this source file and falling back to executables/ in cwd.
func getHallReqAssingerPath() string {
	binaryName := "hall_request_assigner"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "executables", binaryName))
		if _, err := os.Stat(resolvedPath); err == nil {
			return resolvedPath
		}
	}

	return filepath.Clean(filepath.Join("executables", binaryName))
}
