package statesync

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	network "github.com/Mosazghi/elevator-ttk4145/internal/network"
	"github.com/Mosazghi/elevator-ttk4145/pkg/checksum"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"
	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/vmihailenco/msgpack/v5"
)

type Worldview struct {
	LocalID              int                          `json:"local_id"`
	ElevatorStates       map[int]*RemoteElevatorState `json:"elevator_states"`
	HallCalls            [][2]HallCallPairState       `json:"hall_calls"`
	NumFloors            int                          `json:"num_floors"`
	orderUpdateChan      chan Order
	recoveredCabCallChan chan Empty
	hasFetchedCabCalls   bool
	mutex                *sync.RWMutex
	lastNetworkError     time.Time
}

// NewWorldView creates a new instance
func NewWorldView(localID, numFloors int, orderUpdateChan chan Order, recoveredCabCallChan chan Empty) *Worldview {
	wv := &Worldview{
		LocalID:              localID,
		ElevatorStates:       make(map[int]*RemoteElevatorState),
		HallCalls:            make([][2]HallCallPairState, numFloors),
		NumFloors:            numFloors,
		orderUpdateChan:      orderUpdateChan,
		hasFetchedCabCalls:   false,
		recoveredCabCallChan: recoveredCabCallChan,
		mutex:                &sync.RWMutex{},
		lastNetworkError:     time.Time{},
	}

	for i := range wv.HallCalls {
		wv.HallCalls[i][HDDown].AssignedBy = UnassignedID
		wv.HallCalls[i][HDUp].AssignedBy = UnassignedID
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
			wvCopy.HallCalls[i][j] = HallCallPairState{
				State:      call.State,
				AssignedBy: call.AssignedBy,
				Timestamp:  call.Timestamp,
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
func (wv *Worldview) StartSyncing(txChan chan<- network.DataPacket, rxChan <-chan network.DataPacket, netErrChan <-chan error) {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()
	localID := wv.LocalID

	for {
		select {
		case <-netErrChan:
			wv.mu.Lock()
			wv.lastNetworkError = time.Now()
			wv.mu.Unlock()
		case peerData := <-rxChan:
			message := Message{}
			err := msgpack.Unmarshal(peerData, &message)
			if err != nil {
				slog.Error("Failed to unmarshal message", "error", err)
				continue
			}

			otherWv := message.Worldview

			if otherWv.LocalID == localID {
				continue
			}

			wv.mutex.Lock()
			wv.checkifNodeReappeared(otherWv.LocalID)
			wv.mutex.Unlock()

			err = wv.Merge(&otherWv, message.Checksum)
			if err != nil {
				slog.Error("Failed to merge worldview", "error", err)
			}

		case <-ticker.C:
			wv.mutex.Lock()
			wv.ElevatorStates[localID].LastSeenAt = time.Now()
			wv.releaseAnyOrders()
			wv.mutex.Unlock()

			wv.mutex.RLock()
			wvSnapshot := wv.deepCopy()
			wv.mutex.RUnlock()

			data, err := BuildWvJSON(&wvSnapshot)
			if err != nil {
				slog.Error("Failed to build worldview message", "error", err)
				continue
			}

			txChan <- data
		}
	}
}

// fetchCabCallsOnReconnect ORs the local Cab Calls with those of the peer
// and signals cab lights to be turned back on when applicable.
// Must be called with lock held.
func (wv *Worldview) fetchCabCallsOnReconnect(other *Worldview) {
	peerView := other.ElevatorStates[wv.LocalID]

	for floor, peerCabCallValue := range peerView.CabCalls {
		existingCabCallValue := wv.ElevatorStates[wv.LocalID].CabCalls[floor]
		wv.ElevatorStates[wv.LocalID].CabCalls[floor] = existingCabCallValue || peerCabCallValue
	}
	wv.recoveredCabCallChan <- Empty{}
	slog.Debug("cab calls recovered from peer", "cabCalls", wv.ElevatorStates[wv.LocalID].CabCalls)
}

func (wv *Worldview) checkifNodeReappeared(id int) {
	other, exist := wv.ElevatorStates[id]
	if exist && !other.Alive {
		slog.Info("Node has reappeared", "id", id)
		wv.ElevatorStates[id].Alive = true
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

		if time.Since(state.LastSeenAt) > NodeLostTimeout && wv.ElevatorStates[id].Alive {
			slog.Warn("Lost peer", "id", id, "lastSeen", state.LastSeenAt.Format(time.RFC3339))
			wv.ElevatorStates[id].Alive = false
		}
	}

	for floor := range wv.HallCalls {
		for dir := range wv.HallCalls[floor] {
			hc := wv.HallCalls[floor][dir]

			assignedNode, exist := wv.ElevatorStates[hc.AssignedBy]
			isNodeLost := exist && !assignedNode.Alive

			hasOrderTimedout := hc.State == HallCallStateProcessing && hc.Timestamp != 0 &&
				time.Since(time.UnixMilli(hc.Timestamp)) > OrderProcessingTimeout

			if isNodeLost || hasOrderTimedout {
				reason := "unknown"
				switch {
				case isNodeLost:
					reason = "node lost"
				case hasOrderTimedout:
					if hc.AssignedBy == localID {
						wv.ElevatorStates[localID].TimedOutAt = time.Now()
						slog.Warn("unable to complete order, temporarily blocked from new hall assignments.")
					}
					reason = "order timed out"
				}

				slog.Warn("releasing order", "by", hc.AssignedBy, "floor", floor, "dir", dir, "reason", reason)
				wv.HallCalls[floor][dir] = HallCallPairState{
					State:      HallCallStateConfirmed,
					AssignedBy: UnassignedID,
					Timestamp:  0,
				}
			}

		}
	}
}

// isDisconnected returns true if the local node is considered disconnected from the network,
// false otherwise
func (wv *Worldview) isDisconnected() bool {
	return time.Since(wv.lastNetworkError) <= time.Duration(DisconnectedTimeout)
}

// setHallCall changes the given floor's Up/Down state based on dir
func (wv *Worldview) setHallCall(floor int, dir HallCallDir, state HallCallState) error {
	wv.mutex.Lock()
	defer wv.mutex.Unlock()

	if !IsValidFloor(floor, wv.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}
	existingHallCall := wv.HallCalls[floor][dir]

	if err := IsValidDirTransition(existingHallCall.State, state); err != nil {
		return fmt.Errorf("invalid state transition for floor %d dir %d: %w", floor, dir, err)
	}

	var resultHallCall HallCallPairState
	resultHallCall.State = state

	if state == HallCallStateProcessing {
		resultHallCall.AssignedBy = wv.LocalID
		resultHallCall.Timestamp = time.Now().UnixMilli()
	} else {
		resultHallCall.AssignedBy = UnassignedID
		resultHallCall.Timestamp = 0
	}

	if state == HallCallStateNone && existingHallCall.State == HallCallStateProcessing {
		if existingHallCall.AssignedBy != wv.LocalID {
			return fmt.Errorf("cannot complete hall call that is being processed by another elevator")
		}
	}

	wv.HallCalls[floor][dir] = resultHallCall
	return nil
}

// CompleteHallCall marks the given hall call as completed, but only if it is currently being processed by the local elevator
func (wv *Worldview) CompleteHallCall(floor int, dir HallCallDir) error {
	if err := wv.setHallCall(floor, dir, HallCallStateNone); err != nil {
		return err
	}
	wv.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: true}
	return nil
}

// NewHallCall creates a new order on the systems
func (wv *Worldview) NewHallCall(floor int, dir HallCallDir) error {
	wv.mutex.RLock()

	if wv.isDisconnected() {
		wv.mu.RUnlock()
		return fmt.Errorf("cannot place new hall call when network is disconnected")
	}

	aliveCount := 0
	for _, state := range wv.ElevatorStates {
		if state.Alive {
			aliveCount++
		}
	}

	localElevator := wv.ElevatorStates[wv.LocalID]
	if aliveCount <= 1 && localElevator.IsObstructed {
		wv.mutex.RUnlock()
		return fmt.Errorf("cannot place new hall call when alone on network and local elevator is obstructed")
	}

	var targetState HallCallState

	wv.mutex.RUnlock()

	// Alone on network , skip Unconfirmed, go straight to Confirmed
	if aliveCount <= 1 {
		targetState = HallCallStateConfirmed
	} else {
		targetState = HallCallStateUnconfirmed
	}

	if err := wv.setHallCall(floor, dir, targetState); err != nil {
		return err
	}

	wv.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
	return nil
}

// ProcessHallCall process the hall call by the local elevator
func (wv *Worldview) ProcessHallCall(floor int, dir HallCallDir) error {
	return wv.setHallCall(floor, dir, HallCallStateProcessing)
}

// SetCabCall changes cab call state at floor
func (wv *Worldview) SetCabCall(floor int, state bool) error {
	slog.Info("setting cab call", "floor", floor, "state", state)
	wv.mutex.Lock()
	defer wv.mutex.Unlock()

	if !IsValidFloor(floor, wv.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}

	wv.ElevatorStates[wv.LocalID].CabCalls[floor] = state

	wv.orderUpdateChan <- Order{Type: elevio.Cab, Floor: floor, Completed: !state}

	return nil
}

// SetLocalElevator updates the local elevator state in the worldview
func (wv *Worldview) SetLocalElevator(elev *RemoteElevatorState) error {
	wv.mutex.Lock()
	defer wv.mutex.Unlock()
	if err := ValidateStateRemote(elev); err != nil {
		return err
	}

	wv.ElevatorStates[wv.LocalID] = elev
	return nil
}

func (wv *Worldview) SetOtherElevator(elev *RemoteElevatorState, id int) error {
	wv.mutex.Lock()
	defer wv.mutex.Unlock()
	if err := ValidateStateRemote(elev); err != nil {
		return err
	}

	wv.ElevatorStates[id] = elev
	return nil
}

// GetRemoteElevator returns the local elevator state from the worldview
func (wv *Worldview) GetRemoteElevator() RemoteElevatorState {
	wv.mutex.RLock()
	defer wv.mutex.RUnlock()
	return *wv.ElevatorStates[wv.LocalID]
}

// GetAllHallCalls returns a copy of the current hall calls in the worldview
func (wv *Worldview) GetAllHallCalls() [][2]HallCallPairState {
	wv.mutex.RLock()
	defer wv.mutex.RUnlock()

	result := make([][2]HallCallPairState, len(wv.HallCalls))
	copy(result, wv.HallCalls)

	return result
}

// Merge merges incoming Worldview into the current one
func (wv *Worldview) Merge(other *Worldview, otherChecksum uint64) error {
	wv.mutex.Lock()
	defer wv.mutex.Unlock()

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

	_, weExist := other.ElevatorStates[wv.LocalID]

	wv.ElevatorStates[other.LocalID] = otherLocalState

	fetchCabCalls := false

	if !wv.hasFetchedCabCalls && weExist {
		fetchCabCalls = !slices.Equal(wv.ElevatorStates[wv.LocalID].CabCalls, other.ElevatorStates[wv.LocalID].CabCalls)
	}

	if fetchCabCalls {
		wv.fetchCabCallsOnReconnect(other)
		wv.hasFetchedCabCalls = true
	}

	// -- Validate Hall Calls --
	for floor := range other.HallCalls {
		for dir := range other.HallCalls[floor] {
			otherHallCall := other.HallCalls[floor][dir]
			ourHallCall := wv.HallCalls[floor][dir]
			dir := HallCallDir(dir)
			switch otherHallCall.State {
			case HallCallStateNone:
				// We accept even if our is Confirmed because 'other' might have completed from the
				// same floor as the order floor, thus bypassing Processing sequence
				if (ourHallCall.State == HallCallStateProcessing || ourHallCall.State == HallCallStateConfirmed) && ourHallCall.AssignedBy == other.LocalID {
					slog.Info("order completed", "by", ourHallCall.AssignedBy, "floor", floor, "dir", dir, "prevState", ourHallCall.State)

					wv.HallCalls[floor][dir] = HallCallPairState{
						State:      HallCallStateNone,
						AssignedBy: UnassignedID,
						Timestamp:  time.Now().UnixMilli(), // keep timestamp on None to mark "just completed"
					}
					wv.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: true}
				}
			case HallCallStateUnconfirmed:
				if ourHallCall.State == HallCallStateNone {
					slog.Info("new uncofirmed order", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir)
					wv.HallCalls[floor][dir].State = HallCallStateUnconfirmed

					wv.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
				}

				if ourHallCall.State == HallCallStateUnconfirmed {
					slog.Info("order confirmed, promoting to CONFRIMED", "floor", floor, "dir", dir)
					wv.HallCalls[floor][dir].State = HallCallStateConfirmed
				}
			case HallCallStateConfirmed:
				if ourHallCall.State == HallCallStateNone || ourHallCall.State == HallCallStateUnconfirmed {
					// Peer already promoted → accept it
					wv.HallCalls[floor][dir].State = HallCallStateConfirmed

					if ourHallCall.State == HallCallStateNone {
						wv.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
					}
				}

				if ourHallCall.State == HallCallStateProcessing && ourHallCall.AssignedBy == other.LocalID {
					slog.Warn("order has been released", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir)
					wv.HallCalls[floor][dir] = HallCallPairState{
						State:      HallCallStateConfirmed,
						AssignedBy: UnassignedID,
						Timestamp:  0,
					}
				}

			case HallCallStateProcessing:
				// We can accept if our is none because we might be on startup
				if ourHallCall.State == HallCallStateConfirmed || ourHallCall.State == HallCallStateNone {
					if otherHallCall.AssignedBy == other.LocalID {
						slog.Info("processing order", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir, "timestamp", otherHallCall.Timestamp)
						wv.HallCalls[floor][dir].AssignedBy = otherHallCall.AssignedBy
						wv.HallCalls[floor][dir].State = HallCallStateProcessing
						wv.HallCalls[floor][dir].Timestamp = otherHallCall.Timestamp
					}
				}
			}
		}
	}

	return nil
}
