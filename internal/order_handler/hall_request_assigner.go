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
func CalculateCost(worldview *statesync.Worldview, floor int, direction statesync.HallCallDir) (int, error) {
	hrs := BuildHallRequestAssignerData(worldview)
	winnerID := statesync.UnassignedID

	jsonData, err := json.Marshal(hrs)
	if err != nil {
		return winnerID, fmt.Errorf("failed to marshal hall request assigner data: %w", err)
	}

	hraPath := getHallReqAssingerPath()

	execCmd := exec.Command(hraPath, "--input", string(jsonData))
	output, err := execCmd.Output()
	if err != nil {
		return winnerID, fmt.Errorf("failed to execute hall request assigner: %w", err)
	}

	var result HallRequestAssignerOutput

	err = json.Unmarshal(output, &result)
	// the winner is who got true in the corresponding floor and direction
	if err != nil {
		return winnerID, fmt.Errorf("failed to unmarshal hall request assigner output: %w", err)
	}

	for idString, hallCallAssignments := range result {
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

// BuildHallRequestAssignerData converts worldview state to assigner input,
// excluding elevators that are currently not allowed to serve orders.
func BuildHallRequestAssignerData(worldview *statesync.Worldview) *HallRequestAssignerInput {
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
		if !elevator.AllowedToServe() {
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
// relative to this source file and falling back to simulator/ in cwd.
func getHallReqAssingerPath() string {
	binaryName := "hall_request_assigner"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "simulator", binaryName))
		if _, err := os.Stat(resolvedPath); err == nil {
			return resolvedPath
		}
	}

	return filepath.Clean(filepath.Join("simulator", binaryName))
}
