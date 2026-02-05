package statesync

import (
	"fmt"
	"sync"
	"time"

	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
)

type Worldview struct {
	localID             int
	elevatorStates      map[int]RemoteElevatorState
	lostElevatorsState  map[int]RemoteElevatorState
	hallCalls           [][2]HallCallPairState
	syncLocalRemoteChan chan RemoteElevatorState
	localRemoteState    RemoteElevatorState
	numFloors           int
	checksum            uint64
	wvChan              chan Worldview
	mu                  *sync.Mutex
}

func NewWorldView(localID, numFloors int) *Worldview {
	wv := &Worldview{
		localID:             localID,
		elevatorStates:      make(map[int]RemoteElevatorState),
		hallCalls:           make([][2]HallCallPairState, numFloors),
		numFloors:           numFloors,
		syncLocalRemoteChan: make(chan RemoteElevatorState, 10),
		wvChan:              make(chan Worldview),
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

func (wv *Worldview) StartSyncing(port, id int) error {
	return nil
}

// SetHallCall changes the given floor's Up/Down state based on dir
func (wv *Worldview) SetHallCall(floor int, dir HallCallDir, state HallCallState) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if !IsValidFloor(floor, wv.numFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}

	currDirState := wv.hallCalls[floor][dir]

	if err := IsValidDirTransition(currDirState.State, state); err != nil {
		return fmt.Errorf("invalid state transition for floor %d dir %d: %w", floor, dir, err)
	}

	wv.hallCalls[floor][dir] = HallCallPairState{
		State: state,
		By:    wv.localID,
	}

	wv.updateChecksum()

	return nil
}

// SetCabCall changes cab call state at floor
func (wv *Worldview) SetCabCall(floor int, state bool) bool {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if !IsValidFloor(floor, wv.numFloors) {
		return false
	}

	wv.localRemoteState.CabCalls[floor] = state
	wv.updateChecksum()

	return true
}

func (wv *Worldview) SetLocalElevator(remoteElev *RemoteElevatorState) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	return nil
}

func (wv *Worldview) GetAllHallCalls() [][2]HallCallPairState {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	result := make([][2]HallCallPairState, len(wv.hallCalls))
	copy(result, wv.hallCalls)

	return result
}

// Merge merges incoming Worldview into the current one
func (wv *Worldview) Merge(other *Worldview) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if other == nil {
		return fmt.Errorf("cannot merge with nil worldview")
	}

	otherHCLen := len(other.hallCalls)
	ourHCLen := len(wv.hallCalls)

	if otherHCLen > ourHCLen || otherHCLen < ourHCLen {
		return fmt.Errorf("length of hall calls doesnt match")
	}

	if other.numFloors != wv.numFloors {
		return fmt.Errorf("number of floors doesnt match")
	}

	calculatedChecksum, err := checksum.CalculateChecksum(other)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != other.checksum {
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// -- Validate Elevator State --
	remoteElevState := other.elevatorStates[other.localID]
	if !ValidateStateRemote(&remoteElevState) {
		return fmt.Errorf("%v's local state is invalid", other.localID)
	}

	wv.elevatorStates[other.localID] = remoteElevState

	// NOTE!: Approp. place to clean up old elevator states?
	for id, state := range wv.elevatorStates {
		if time.Since(state.LastSeenAt) > 5*time.Second {
			delete(wv.elevatorStates, id)
		}
	}

	// -- Validate Hall Calls --
	// Merge hall calls
	for floor := range other.hallCalls {
		for dir := range other.hallCalls[floor] {
			otherDirState := other.hallCalls[floor][dir]
			ourDirState := wv.hallCalls[floor][dir]

			// 1. Check if others has fullfilled the call
			if otherDirState.State == HSNone && ourDirState.State == HSProcessing && other.localID == otherDirState.By {
				wv.hallCalls[floor][dir] = otherDirState
			}

			// 2. Check if others has received a new order
			if otherDirState.State == HSAvailable {
				if ourDirState.State == HSNone {
					wv.hallCalls[floor][dir] = otherDirState
				}
			}

			// 3. Check if others is processing an order
			// though, it can only go from available -> processing
			if otherDirState.State == HSProcessing && ourDirState.State == HSAvailable {
				wv.hallCalls[floor][dir] = otherDirState
			}

			// NOTE!: For later if we need to release the order [Release active stale orders taken by disconnected elevators]
			// if otherDirState.State == HSProcessing && ourDirState.State == HSProcessing {
			// 	if _, exists := wv.elevatorStates[otherDirState.By]; !exists {
			// 		wv.hallCalls[floor][dir] = HallCallPairState{
			// 			State: HSAvailable,
			// 			By:    0,
			// 		}
			// 	}
			// }
		}
	}

	return nil
}

// updateChecksum recalculates the worldview's checksum
func (wv *Worldview) updateChecksum() error {
	cs, _ := checksum.CalculateChecksum(wv)
	wv.checksum = cs

	return nil
}

func (wv *Worldview) UpdateLocalElevatorFloor(floor int) {
	wv.localRemoteState.CurrentFloor = floor
}

func (wv *Worldview) UpdateLocalElevatorBehavior(behavior elevator.Behavior) {
	wv.localRemoteState.Behavior = behavior
}
