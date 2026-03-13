package orders

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	statesync "github.com/Mosazghi/elevator-ttk4145/internal/statesync"
)

type HallRequestAssignerElevState struct {
	Behavior    string `json:"behavior"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HallRequestAssignerInput struct {
	HallRequests [][2]bool                               `json:"hallRequests"`
	States       map[string]HallRequestAssignerElevState `json:"states"`
}

type HallRequestAssignerOutput map[string][][2]bool

func CalculateCost(wv *statesync.Worldview, floor int, dir statesync.HallCallDir) ElevatorCost {
	winner := ElevatorCost{-1, wv.NumFloors + 1, 100}
	hrs := BuildHallRequestAssignerData(wv)

	jsonData, err := json.Marshal(hrs)
	if err != nil {
		slog.Error("failed to marshal hall request assigner data", "error", err)
		return winner
	}

	hraPath := getHallReqAssingerPath()

	execCmd := exec.Command(hraPath, "--input", string(jsonData))
	output, err := execCmd.Output()
	if err != nil {
		slog.Error("failed to execute hall request assigner", "error", err)
		return winner
	}

	var result HallRequestAssignerOutput

	err = json.Unmarshal(output, &result)
	// the winner is who got true in the corresponding floor and direction
	if err != nil {
		slog.Error("failed to unmarshal hall request assigner output", "error", err)
		return winner
	}
	log.Println("hall request assigner output", "result", result)

	for id, assigned := range result {
		if assigned[floor][dir] {
			ID, err := strconv.Atoi(id)
			if err != nil {
				slog.Error("failed to parse elevator ID from hall request assigner output", "error", err, "id", id)
				continue
			}
			winner.id = ID
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
		States:       make(map[string]HallRequestAssignerElevState),
	}

	// Build hall calls
	for floor, pair := range hcs {
		for dir, state := range pair {
			isActive := state.State == statesync.HSAvailable || state.State == statesync.HSProcessing
			if isActive {
				hrs.HallRequests[floor][dir] = true
			}
		}
	}

	// Build elevator states
	for id, elev := range wv.ElevatorStates {
		if !elev.Alive || elev.IsObstructed {
			continue
		}

		hrs.States[strconv.Itoa(id)] = HallRequestAssignerElevState{
			Behavior:    elev.Behavior.String(),
			Floor:       elev.CurrentFloor,
			Direction:   elev.Direction.String(),
			CabRequests: elev.CabCalls,
		}
	}

	return &hrs
}

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
