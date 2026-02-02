package statesync

import (
	"fmt"
	"sync"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
)

type HallCallState int

const (
	HSNone HallCallState = iota
	HSAvailable
	HSProcessing
)

type HallCallPairState struct {
	State HallCallState
	By    int
}

type HallCallPair struct {
	Up   HallCallPairState
	Down HallCallPairState
}

type Worldview struct {
	localID             int
	elevatorStates      map[int]RemoteElevatorState
	hallCalls           []HallCallPair
	syncLocalRemoteChan chan RemoteElevatorState
	localRemoteState    RemoteElevatorState
	numFloors           int
	checksum            uint64
	mu                  *sync.Mutex
}

func NewTestWorldview(localID, numFloors int) *Worldview {
	wv := &Worldview{
		localID:             localID,
		elevatorStates:      make(map[int]RemoteElevatorState),
		hallCalls:           make([]HallCallPair, numFloors),
		numFloors:           numFloors,
		syncLocalRemoteChan: make(chan RemoteElevatorState, 10),
		localRemoteState: RemoteElevatorState{
			ID:           localID,
			CurrentFloor: 0,
			Direction:    elevio.MDUp,
			DoorState:    elevator.DSClosed,
			CabCalls:     make([]bool, numFloors),
			Behavior:     elevator.BIdle,
			LastSeenAt:   time.Now(),
		},
		mu: &sync.Mutex{},
	}
	wv.updateChecksum()
	return wv
}

// Helper to update checksum after modifying worldview
func (wv *Worldview) updateChecksum() error {
	cs, _ := checksum.CalculateChecksum(wv)
	wv.checksum = cs

	return nil
}

// / Merge merges another Worldview into the current one.
// / It does some sanity checks to ensure data integrity.
// /
func (wv *Worldview) Merge(other *Worldview) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if other == nil {
		return fmt.Errorf("cannot merge with nil worldview")
	}

	if wv.numFloors != other.numFloors {
		return fmt.Errorf("cannot merge worldviews with different number of floors")
	}

	calculatedChecksum, err := checksum.CalculateChecksum(other)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != other.checksum {
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// Merge hall calls
	for floor, callPair := range other.hallCalls {
		// 1. Check if others has fullfilled the call
		if callPair.Up.State == HSNone && other.localID == callPair.Up.By {
			wv.hallCalls[floor].Up = callPair.Up
		}

		if callPair.Down.State == HSNone && other.localID == callPair.Down.By {
			wv.hallCalls[floor].Down = callPair.Down
		}

		// 2. Check if others has received a new order
		if callPair.Up.State == HSAvailable {
			if wv.hallCalls[floor].Up.State == HSNone {
				wv.hallCalls[floor].Up = callPair.Up
			}
		}

		if callPair.Down.State == HSAvailable {
			if wv.hallCalls[floor].Down.State == HSNone {
				wv.hallCalls[floor].Down = callPair.Down
			}
		}

		// 3. Check if others is processing an order
		// though, it can only go from available -> processing
		if callPair.Up.State == HSProcessing && wv.hallCalls[floor].Up.State == HSAvailable {
			wv.hallCalls[floor].Up = callPair.Up
		}

		if callPair.Down.State == HSProcessing && wv.hallCalls[floor].Down.State == HSAvailable {
			wv.hallCalls[floor].Down = callPair.Down
		}

	}

	return nil
}

func (wv *Worldview) AddElevator(elevator RemoteElevatorState) {}
func (wv *Worldview) AddHallCall(floor int, call HallCallPair) {}

func (wv *Worldview) GetElevators() map[int]RemoteElevatorState {
	return nil
}

func (wv *Worldview) GetHallCalls() []HallCallPair {
	return nil
}
