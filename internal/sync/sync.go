package statesync

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
)

type Message struct {
	Wv       Worldview
	Checksum uint64
}

type Worldview struct {
	LocalID             int                          `json:"local_id"`
	ElevatorStates      map[int]*RemoteElevatorState `json:"elevator_states"`
	lostElevatorsState  map[int]*RemoteElevatorState
	HallCalls           [][2]HallCallPairState `json:"hall_calls"`
	syncLocalRemoteChan chan RemoteElevatorState
	// localRemoteState    *RemoteElevatorState
	NumFloors int `json:"num_floors"`
	// checksum            uint64
	wvChan chan Worldview
	mu     *sync.Mutex
}

// NewWorldView creates a new instance
func NewWorldView(localID, numFloors int) *Worldview {
	wv := &Worldview{
		LocalID:             localID,
		ElevatorStates:      make(map[int]*RemoteElevatorState),
		lostElevatorsState:  make(map[int]*RemoteElevatorState),
		HallCalls:           make([][2]HallCallPairState, numFloors),
		NumFloors:           numFloors,
		syncLocalRemoteChan: make(chan RemoteElevatorState, 10),
		wvChan:              make(chan Worldview),
		// localRemoteState:    NewRemoteElevatorState(localID, numFloors),
		mu: &sync.Mutex{},
	}
	wv.ElevatorStates[localID] = NewRemoteElevatorState(localID, numFloors)

	return wv
}

func (wv Worldview) String() string {

	return fmt.Sprintf("Worldview{LocalID: %d, ElevatorStates: %v, HallCalls: %v, NumFloors: %d}",
		wv.LocalID, wv.ElevatorStates, wv.HallCalls, wv.NumFloors)
}

// StartSyncing creates listeners and transmitters for synchroizations with other elevators
func (wv *Worldview) StartSyncing(txChan chan<- network.UDPMessage, rxChan <-chan network.UDPMessage, errChan <-chan error) {

	ticker := time.NewTicker(BroadcastInterval)
	localID := wv.LocalID
	for {
		select {
		case err := <-errChan:
			slog.Error("Network error", "error", err)
		case peerData := <-rxChan:
			message := Message{}
			err := json.Unmarshal(peerData.Data, &message)
			if err != nil {
				slog.Error("Failed to unmarshal message", "error", err)
				continue
			}

			otherWv := message.Wv

			if otherWv.LocalID == localID {
				// fmt.Println("Received own broadcast, ignoring...")
				continue
			}
			if err != nil {
				slog.Error("Failed to unmarshal worldview", "error", err)
				continue
			}

			_, exists := wv.lostElevatorsState[otherWv.LocalID]
			if exists {
				slog.Warn("Reappeared peer", "id", otherWv.LocalID)
				delete(wv.lostElevatorsState, otherWv.LocalID)
			}

			err = wv.Merge(&otherWv, message.Checksum)
			if err != nil {
				slog.Error("Failed to merge worldview", "error", err)
				continue
			}

			// fmt.Println("Peerdata: ", otherWv)
		case <-ticker.C:
			wv.mu.Lock()
			wv.ElevatorStates[localID].LastSeenAt = time.Now()

			// NOTE!: Approp. place to clean up old elevator states?
			for id, state := range wv.ElevatorStates {
				if id != wv.LocalID && time.Since(state.LastSeenAt) > NodeTimeoutDelay {
					slog.Warn("Lost peer", "id", id)

					wv.lostElevatorsState[id] = state
					delete(wv.ElevatorStates, id)
				}
			}

			checksum, err := checksum.CalculateChecksum(wv)
			if err != nil {
				slog.Error("Failed to calculate checksum", "error", err)
				wv.mu.Unlock()
				continue
			}
			msg := Message{
				Wv:       *wv,
				Checksum: checksum,
			}

			jsonData, err := json.Marshal(msg)
			if err != nil {
				slog.Error("Failed to marshal worldview", "error", err)
				wv.mu.Unlock()
				continue
			}
			// NOTE!: We can optimize by only sending diffs instead of the whole worldview
			txChan <- network.UDPMessage{
				Data:    jsonData,
				Address: nil, // Broadcast address is handled by the network layer
			}

			wv.mu.Unlock()

		}
	}
}

func (wv *Worldview) BroadCast() {
}

// SetHallCall changes the given floor's Up/Down state based on dir
func (wv *Worldview) SetHallCall(floor int, dir HallCallDir, state HallCallState) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if !IsValidFloor(floor, wv.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}

	currDirState := wv.HallCalls[floor][dir]

	if err := IsValidDirTransition(currDirState.State, state); err != nil {
		return fmt.Errorf("invalid state transition for floor %d dir %d: %w", floor, dir, err)
	}

	wv.HallCalls[floor][dir] = HallCallPairState{
		State: state,
		By:    wv.LocalID,
	}

	return nil
}

// SetCabCall changes cab call state at floor
func (wv *Worldview) SetCabCall(floor int, state bool) bool {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if !IsValidFloor(floor, wv.NumFloors) {
		return false
	}

	wv.ElevatorStates[wv.LocalID].CabCalls[floor] = state

	return true
}

// SetLocalElevator updates the local elevator state in the worldview
func (wv *Worldview) SetLocalElevator(elev *RemoteElevatorState) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	if err := ValidateStateRemote(elev); err != nil {
		return err
	}

	wv.ElevatorStates[wv.LocalID] = elev
	return nil
}

// GetRemoteElevator returns the local elevator state from the worldview
func (wv *Worldview) GetRemoteElevator() RemoteElevatorState {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	return *wv.ElevatorStates[wv.LocalID]
}

// GetAllHallCalls returns a copy of the current hall calls in the worldview
func (wv *Worldview) GetAllHallCalls() [][2]HallCallPairState {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	result := make([][2]HallCallPairState, len(wv.HallCalls))
	copy(result, wv.HallCalls)

	return result
}

// Merge merges incoming Worldview into the current one
func (wv *Worldview) Merge(other *Worldview, otherChecksum uint64) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if other == nil {
		return fmt.Errorf("cannot merge with nil worldview")
	}

	otherHCLen := len(other.HallCalls)
	ourHCLen := len(wv.HallCalls)

	if otherHCLen > ourHCLen || otherHCLen < ourHCLen {
		return fmt.Errorf("length of hall calls doesnt match")
	}

	if other.NumFloors != wv.NumFloors {
		return fmt.Errorf("number of floors doesnt match")
	}

	calculatedChecksum, err := checksum.CalculateChecksum(other)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != otherChecksum {
		fmt.Printf("Calculated checksum: %x, Received checksum: %x\n", calculatedChecksum, otherChecksum)
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// -- Validate Elevator State --
	slog.Debug("validating", "other", other)
	otherLocalState := other.ElevatorStates[other.LocalID]
	if err = ValidateStateRemote(otherLocalState); err != nil {
		return fmt.Errorf("%v's local state is invalid: %w", other.LocalID, err)
	}

	wv.ElevatorStates[other.LocalID] = otherLocalState

	// -- Validate Hall Calls --
	// Merge hall calls
	for floor := range other.HallCalls {
		for dir := range other.HallCalls[floor] {
			otherDirState := other.HallCalls[floor][dir]
			ourDirState := wv.HallCalls[floor][dir]

			// 1. Check if others has fullfilled the call
			if otherDirState.State == HSNone && ourDirState.State == HSProcessing && other.LocalID == otherDirState.By {
				wv.HallCalls[floor][dir] = otherDirState
			}

			// 2. Check if others has received a new order
			if otherDirState.State == HSAvailable {
				if ourDirState.State == HSNone {
					wv.HallCalls[floor][dir] = otherDirState
				}
			}

			// 3. Check if others is processing an order
			// though, it can only go from available -> processing
			if otherDirState.State == HSProcessing && ourDirState.State == HSAvailable {
				wv.HallCalls[floor][dir] = otherDirState
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
// func (wv *Worldview) updateChecksum() error {
// 	cs, err := checksum.CalculateChecksum(wv)
// 	if err != nil {
// 		return err
// 	}

// 	wv.checksum = cs

// 	return nil
// }
