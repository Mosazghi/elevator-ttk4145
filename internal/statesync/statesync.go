package statesync

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	network "github.com/Mosazghi/elevator-ttk4145/internal/net"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
	"github.com/vmihailenco/msgpack/v5"
)

type Message struct {
	Wv       Worldview
	Checksum uint64
}

type Worldview struct {
	LocalID            int                          `json:"local_id"`
	ElevatorStates     map[int]*RemoteElevatorState `json:"elevator_states"`
	lostElevatorsState map[int]*RemoteElevatorState
	HallCalls          [][2]HallCallPairState `json:"hall_calls"`
	NumFloors          int                    `json:"num_floors"`
	wvChan             chan Worldview
	mu                 *sync.RWMutex
}

// NewWorldView creates a new instance
func NewWorldView(localID, numFloors int, wvChan chan Worldview) *Worldview {
	wv := &Worldview{
		LocalID:            localID,
		ElevatorStates:     make(map[int]*RemoteElevatorState),
		lostElevatorsState: make(map[int]*RemoteElevatorState),
		HallCalls:          make([][2]HallCallPairState, numFloors),
		NumFloors:          numFloors,
		wvChan:             wvChan,
		mu:                 &sync.RWMutex{},
	}

	for i := range wv.HallCalls {
		wv.HallCalls[i][HDDown].ConfirmedBy = make([]int, 0)
		wv.HallCalls[i][HDUp].ConfirmedBy = make([]int, 0)
		wv.HallCalls[i][HDDown].By = -1
		wv.HallCalls[i][HDUp].By = -1
	}

	wv.ElevatorStates[localID] = NewRemoteElevatorState(localID, numFloors)

	return wv
}

// deepCopy creates a deep copy of the worldview for safe concurrent marshaling
// Must be called with read lock (RLock) held
// NOTE: when adding new fields, make sure to include them in the deep copy as well.
func (wv *Worldview) deepCopy() Worldview {
	wvCopy := Worldview{
		LocalID:   wv.LocalID,
		NumFloors: wv.NumFloors,
	}

	// Deep copy ElevatorStates map
	wvCopy.ElevatorStates = make(map[int]*RemoteElevatorState, len(wv.ElevatorStates))
	for id, state := range wv.ElevatorStates {
		stateCopy := *state
		stateCopy.CabCalls = make([]bool, len(state.CabCalls))
		copy(stateCopy.CabCalls, state.CabCalls)
		wvCopy.ElevatorStates[id] = &stateCopy
	}

	// Deep copy HallCalls
	wvCopy.HallCalls = make([][2]HallCallPairState, len(wv.HallCalls))
	for i, floorCalls := range wv.HallCalls {
		for j, call := range floorCalls {
			confirmedCopy := make([]int, len(call.ConfirmedBy))
			copy(confirmedCopy, call.ConfirmedBy)
			wvCopy.HallCalls[i][j] = HallCallPairState{
				State:       call.State,
				By:          call.By,
				ConfirmedBy: confirmedCopy,
				Timestamp:   call.Timestamp,
			}
		}
	}

	return wvCopy
}

// String converts worldview into string format
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
			err := msgpack.Unmarshal(peerData.Data, &message)
			if err != nil {
				slog.Error("Failed to unmarshal message", "error", err)
				continue
			}

			otherWv := message.Wv

			if otherWv.LocalID == localID {
				continue
			}

			wv.mu.Lock()
			wv.checkHasNodeReappeared(otherWv.LocalID)
			wv.mu.Unlock()

			err = wv.Merge(&otherWv, message.Checksum)
			if err != nil {
				slog.Error("Failed to merge worldview", "error", err)
			}

		case <-ticker.C:
			wv.mu.Lock()
			wv.ElevatorStates[localID].LastSeenAt = time.Now()
			wv.releaseAnyOrders()
			wv.mu.Unlock()

			wv.mu.RLock()
			wvSnapshot := wv.deepCopy()
			wv.mu.RUnlock()

			data, err := BuildWvJSON(&wvSnapshot)
			if err != nil {
				slog.Error("Failed to build worldview message", "error", err)
				continue
			}

			txChan <- network.UDPMessage{
				Data:    data,
				Address: nil,
			}

		}
	}
}

// checkHasNodeReappeared deletes from `lostElevatorsState` if a node with given id has reappeared
func (wv *Worldview) checkHasNodeReappeared(id int) {
	_, exists := wv.lostElevatorsState[id]
	if exists {
		slog.Info("Reappeared peer", "id", id)
		delete(wv.lostElevatorsState, id)
	}
}

// releaseAnyOrders releases orders that are assigned to lost
// or obstructed nodes,
// or orders that have been processing for too long without completion.
// At release, it immediately signs off its id for confirmation
func (wv *Worldview) releaseAnyOrders() {
	localID := wv.LocalID

	for id, state := range wv.ElevatorStates {
		if id == localID {
			continue
		}

		if time.Since(state.LastSeenAt) > NodeLostTimeout {
			slog.Warn("Lost peer", "id", id, "lastSeen", state.LastSeenAt.Format(time.RFC3339))
			wv.lostElevatorsState[id] = state
			delete(wv.ElevatorStates, id)
		}
	}

	for floor := range wv.HallCalls {
		for dir := range wv.HallCalls[floor] {
			hc := wv.HallCalls[floor][dir]

			assignedNode, aNExists := wv.ElevatorStates[hc.By]
			isObstructed := aNExists && assignedNode.IsObstructed

			_, isNodeLost := wv.lostElevatorsState[hc.By]

			hasOrderTimedout := hc.State == HSProcessing && hc.Timestamp != 0 &&
				time.Since(time.UnixMilli(hc.Timestamp)) > OrderProcessingTimeout && hc.By == wv.LocalID

			if isNodeLost || hasOrderTimedout || isObstructed {
				reason := "unknown"
				switch {
				case isNodeLost:
					reason = "node lost"
				case hasOrderTimedout:
					reason = "order timed out"
				case isObstructed:
					reason = "node obstructed"
				}

				slog.Warn("releasing order", "by", hc.By, "floor", floor, "dir", dir, "reason", reason)
				wv.HallCalls[floor][dir] = HallCallPairState{
					State:       HSAvailable,
					By:          -1,
					ConfirmedBy: []int{wv.LocalID},
					Timestamp:   0,
				}
			}

		}
	}

}

// setHallCall changes the given floor's Up/Down state based on dir
func (wv *Worldview) setHallCall(floor int, dir HallCallDir, state HallCallState) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	// slog.Info("[setHallCall] Setting hall call", "floor", floor, "dir", dir, "state", state) // log the new state of the hall call

	if !IsValidFloor(floor, wv.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}
	currDirState := wv.HallCalls[floor][dir]

	if err := IsValidDirTransition(currDirState.State, state); err != nil {
		return fmt.Errorf("invalid state transition for floor %d dir %d: %w", floor, dir, err)
	}

	existing := wv.HallCalls[floor][dir]

	by := -1
	timestamp := int64(0)
	if state == HSProcessing {
		existing.By = wv.LocalID
		existing.By = wv.LocalID
		by = wv.LocalID
		timestamp = time.Now().UnixMilli()
	}

	confirmedBy := []int{}

	switch state {
	case HSAvailable:
		confirmedBy = append(confirmedBy, wv.LocalID)
	case HSProcessing:
		if existing.By != wv.LocalID {
			return fmt.Errorf("cannot process hall call that is not assigned to local elevator")
		}
		confirmedBy = make([]int, len(existing.ConfirmedBy))
		copy(confirmedBy, existing.ConfirmedBy)
	case HSNone:
	default:
	}

	wv.HallCalls[floor][dir] = HallCallPairState{
		State:       state,
		By:          by,
		ConfirmedBy: confirmedBy,
		Timestamp:   timestamp,
	}

	return nil
}

// CompleteHallCall marks the given hall call as completed, but only if it is currently being processed by the local elevator
func (wv *Worldview) CompleteHallCall(floor int, dir HallCallDir) error {
	return wv.setHallCall(floor, dir, HSNone)
}

// NewHallCall creates a new order on the systems
func (wv *Worldview) NewHallCall(floor int, dir HallCallDir) error {
	return wv.setHallCall(floor, dir, HSAvailable)
}

// ProcessHallCall process the hall call by the local elevator
func (wv *Worldview) ProcessHallCall(floor int, dir HallCallDir) error {
	return wv.setHallCall(floor, dir, HSProcessing)
}

// SetCabCall changes cab call state at floor
func (wv *Worldview) SetCabCall(floor int, state bool) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	if !IsValidFloor(floor, wv.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}

	wv.ElevatorStates[wv.LocalID].CabCalls[floor] = state

	return nil
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

func (wv *Worldview) SetOtherElevator(elev *RemoteElevatorState, id int) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	if err := ValidateStateRemote(elev); err != nil {
		return err
	}

	wv.ElevatorStates[id] = elev
	return nil
}

// GetRemoteElevator returns the local elevator state from the worldview
func (wv *Worldview) GetRemoteElevator() RemoteElevatorState {
	wv.mu.RLock()
	defer wv.mu.RUnlock()
	return *wv.ElevatorStates[wv.LocalID]
}

// GetAllHallCalls returns a copy of the current hall calls in the worldview
func (wv *Worldview) GetAllHallCalls() [][2]HallCallPairState {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

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

	if otherHCLen != ourHCLen {
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
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// -- Validate Elevator State --
	otherLocalState := other.ElevatorStates[other.LocalID]
	if err = ValidateStateRemote(otherLocalState); err != nil {
		return fmt.Errorf("%v's local state is invalid: %w", other.LocalID, err)
	}
	wv.ElevatorStates[other.LocalID] = otherLocalState

	//
	slog.Debug("hc[0]", "0", wv.HallCalls[0])
	slog.Debug("hc[3]", "3", wv.HallCalls[3])

	// -- Validate Hall Calls --
	// Merge hall calls
	for floor := range other.HallCalls {
		for dir := range other.HallCalls[floor] {
			otherHCState := other.HallCalls[floor][dir]
			ourHCState := wv.HallCalls[floor][dir]
			switch otherHCState.State {
			case HSNone:
				if ourHCState.State == HSProcessing && otherHCState.By == other.LocalID {
					slog.Info("order completed", "by", otherHCState.By, "floor", floor, "dir", dir, "prevState", ourHCState.State)

					wv.HallCalls[floor][dir] = HallCallPairState{
						State:       HSNone,
						By:          -1,
						ConfirmedBy: []int{},
						Timestamp:   0,
					}
				}
			case HSAvailable:
				for _, id := range otherHCState.ConfirmedBy {
					_, isLost := wv.lostElevatorsState[id]
					_, isAlive := wv.ElevatorStates[id]
					if !slices.Contains(wv.HallCalls[floor][dir].ConfirmedBy, id) && isAlive && !isLost {
						wv.HallCalls[floor][dir].ConfirmedBy = append(wv.HallCalls[floor][dir].ConfirmedBy, id)
					}
				}

				if ourHCState.State == HSNone {
					slog.Info("[Merge] new order", "by", otherHCState.By, "floor", floor, "dir", dir, "by", otherHCState.By)
					wv.HallCalls[floor][dir].State = HSAvailable
					if !slices.Contains(wv.HallCalls[floor][dir].ConfirmedBy, wv.LocalID) {
						slog.Info("[Merge] confirming order", "floor", floor, "dir", dir)
						wv.HallCalls[floor][dir].ConfirmedBy = append(wv.HallCalls[floor][dir].ConfirmedBy, wv.LocalID)

					}
				}

				if ourHCState.State == HSProcessing && ourHCState.By == other.LocalID {
					slog.Warn("order has been released", "by", otherHCState.By, "floor", floor, "dir", dir)
					wv.HallCalls[floor][dir] = HallCallPairState{
						State:       HSAvailable,
						By:          -1,
						ConfirmedBy: []int{wv.LocalID},
						Timestamp:   0,
					}
				}
			case HSProcessing:
				if ourHCState.State == HSAvailable && otherHCState.By == other.LocalID {
					slog.Info("processing order", "by", otherHCState.By, "floor", floor, "dir", dir, "timestamp", otherHCState.Timestamp)
					wv.HallCalls[floor][dir].By = otherHCState.By
					wv.HallCalls[floor][dir].State = HSProcessing
					wv.HallCalls[floor][dir].Timestamp = otherHCState.Timestamp
				}

			}
		}
	}

	// Non-blocking send to wvChan, if no-one is listening, we skip sending the update
	select {
	case wv.wvChan <- *wv:
	default:
	}

	return nil
}
