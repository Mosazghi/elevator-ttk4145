// Package statesync provides mechanisms and helper functions to facilitate proper synchronization between peers.
package statesync

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	network "github.com/Mosazghi/elevator-ttk4145/internal/network"
	"github.com/Mosazghi/elevator-ttk4145/pkg/checksum"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	. "github.com/Mosazghi/elevator-ttk4145/pkg/shared"
	"github.com/vmihailenco/msgpack/v5"
)

// Worldview represents the local elevator's view of the system, including its own state and the states of other elevators.
type Worldview struct {
	// Shared state
	LocalID        int                          `json:"local_id"`
	ElevatorStates map[int]*RemoteElevatorState `json:"elevator_states"`
	HallCalls      [][2]HallCallPairState       `json:"hall_calls"`
	NumFloors      int                          `json:"num_floors"`
	// Internal state
	orderUpdateChan      chan Order
	recoveredCabCallChan chan Empty
	hasFetchedCabCalls   bool
	mutex                *sync.RWMutex
	lastNetworkError     time.Time
}

// NewWorldView constructs a new instance of the WorldView struct.
func NewWorldView(localID, numFloors int, orderUpdateChan chan Order, recoveredCabCallChan chan Empty) *Worldview {
	worldview := &Worldview{
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

	for i := range worldview.HallCalls {
		worldview.HallCalls[i][HallCallDirectionDown].AssignedBy = UnassignedID
		worldview.HallCalls[i][HallCallDirectionUp].AssignedBy = UnassignedID
	}

	worldview.ElevatorStates[localID] = NewRemoteElevatorState(localID, numFloors)

	return worldview
}

// deepCopy creates a copy of the worldview for safe concurrent marshaling
// Must be called with read lock (RLock) held
// NOTE: when adding new fields, make sure to include them in the deep copy as well.
func (worldview *Worldview) deepCopy() Worldview {
	wvCopy := Worldview{
		LocalID:   worldview.LocalID,
		NumFloors: worldview.NumFloors,
	}

	// Deep copy ElevatorStates map
	wvCopy.ElevatorStates = make(map[int]*RemoteElevatorState, len(worldview.ElevatorStates))
	for id, state := range worldview.ElevatorStates {
		stateCopy := *state
		stateCopy.CabCalls = make([]bool, len(state.CabCalls))
		copy(stateCopy.CabCalls, state.CabCalls)
		wvCopy.ElevatorStates[id] = &stateCopy
	}

	// Deep copy HallCalls
	wvCopy.HallCalls = make([][2]HallCallPairState, len(worldview.HallCalls))
	for i, floorCalls := range worldview.HallCalls {
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

// String converts the worldview into a string format
func (worldview Worldview) String() string {
	return fmt.Sprintf("Worldview{LocalID: %d, ElevatorStates: %v, HallCalls: %v, NumFloors: %d}",
		worldview.LocalID, worldview.ElevatorStates, worldview.HallCalls, worldview.NumFloors)
}

// StartSyncing creates listeners and transmitters for synchroization with other elevators.
// Incoming data from peers is synchronized and merged upon reception.
// Current states are passed along after successful merging.
func (worldview *Worldview) StartSyncing(txChan chan<- network.DataPacket, rxChan <-chan network.DataPacket, netErrChan <-chan error) {
	ticker := time.NewTicker(broadcastInterval)
	defer ticker.Stop()
	localID := worldview.LocalID

	for {
		select {
		case <-netErrChan:
			worldview.mutex.Lock()
			worldview.lastNetworkError = time.Now()
			worldview.mutex.Unlock()
		case peerData := <-rxChan:
			message := Message{}
			err := msgpack.Unmarshal(peerData, &message)
			if err != nil {
				slog.Error("Failed to unmarshal message", "error", err)
				continue
			}

			otherWorldview := message.Worldview

			if otherWorldview.LocalID == localID {
				continue
			}

			worldview.mutex.Lock()
			worldview.checkifNodeReappeared(otherWorldview.LocalID)
			worldview.mutex.Unlock()

			err = worldview.Merge(&otherWorldview, message.Checksum)
			if err != nil {
				slog.Error("Failed to merge worldview", "error", err)
			}

		case <-ticker.C:
			worldview.mutex.Lock()
			worldview.ElevatorStates[localID].LastSeenAt = time.Now()
			worldview.releaseAnyOrders()
			worldview.mutex.Unlock()

			worldview.mutex.RLock()
			wvSnapshot := worldview.deepCopy()
			worldview.mutex.RUnlock()

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
func (worldview *Worldview) fetchCabCallsOnReconnect(other *Worldview) {
	peerView := other.ElevatorStates[worldview.LocalID]

	for floor, peerCabCallValue := range peerView.CabCalls {
		existingCabCallValue := worldview.ElevatorStates[worldview.LocalID].CabCalls[floor]
		worldview.ElevatorStates[worldview.LocalID].CabCalls[floor] = existingCabCallValue || peerCabCallValue
	}

	worldview.recoveredCabCallChan <- Empty{}
}

// checkifNodeReappeared checks it's alive status as registered
// in the incoming worldview and updates it to true if applicable.
func (worldview *Worldview) checkifNodeReappeared(id int) {
	other, exist := worldview.ElevatorStates[id]
	if exist && !other.Alive {
		slog.Info("Node has reappeared", "id", id)
		worldview.ElevatorStates[id].Alive = true
	}
}

// releaseAnyOrders releases orders that are assigned to lost nodes
// obstructed nodes, or orders timed out (took too long to complete).
// Must be called with lock held.
func (worldview *Worldview) releaseAnyOrders() {
	localID := worldview.LocalID

	for id, state := range worldview.ElevatorStates {
		if id == localID {
			continue
		}

		if time.Since(state.LastSeenAt) > nodeLostTimeout && worldview.ElevatorStates[id].Alive {
			slog.Warn("Lost peer", "id", id, "lastSeen", state.LastSeenAt.Format(time.RFC3339))
			worldview.ElevatorStates[id].Alive = false
		}
	}

	for floor := range worldview.HallCalls {
		for direction := range worldview.HallCalls[floor] {
			hallCall := worldview.HallCalls[floor][direction]

			assignedNode, exist := worldview.ElevatorStates[hallCall.AssignedBy]
			isNodeLost := exist && !assignedNode.Alive

			hasOrderTimedout := hallCall.State == HallCallStateProcessing && hallCall.Timestamp != 0 &&
				time.Since(time.UnixMilli(hallCall.Timestamp)) > orderProcessingTimeout

			if isNodeLost || hasOrderTimedout {
				reason := "unknown"
				switch {
				case isNodeLost:
					reason = "node lost"
				case hasOrderTimedout:
					if hallCall.AssignedBy == localID {
						worldview.ElevatorStates[localID].TimedOutAt = time.Now()
						slog.Warn("unable to complete order, temporarily blocked from new hall assignments.")
					}
					reason = "order timed out"
				}

				slog.Warn("releasing order", "by", hallCall.AssignedBy, "floor", floor, "dir", direction, "reason", reason)
				worldview.HallCalls[floor][direction] = HallCallPairState{
					State:      HallCallStateConfirmed,
					AssignedBy: UnassignedID,
					Timestamp:  0,
				}
			}

		}
	}
}

// isDisconnected returns true if the local node is
// considered disconnected from the network, false otherwise
func (worldview *Worldview) isDisconnected() bool {
	return time.Since(worldview.lastNetworkError) <= time.Duration(disconnectedTimeout)
}

// setHallCall tries to change the given floor's Up/Down state based on direction.
func (worldview *Worldview) setHallCall(floor int, dir HallCallDirection, state HallCallState) error {
	worldview.mutex.Lock()
	defer worldview.mutex.Unlock()

	if !IsValidFloor(floor, worldview.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}
	existingHallCall := worldview.HallCalls[floor][dir]

	if err := IsValidDirTransition(existingHallCall.State, state); err != nil {
		return fmt.Errorf("invalid state transition for floor %d dir %d: %w", floor, dir, err)
	}

	var resultHallCall HallCallPairState
	resultHallCall.State = state

	if state == HallCallStateProcessing {
		resultHallCall.AssignedBy = worldview.LocalID
		resultHallCall.Timestamp = time.Now().UnixMilli()
	} else {
		resultHallCall.AssignedBy = UnassignedID
		resultHallCall.Timestamp = 0
	}

	if state == HallCallStateNone && existingHallCall.State == HallCallStateProcessing {
		if existingHallCall.AssignedBy != worldview.LocalID {
			return fmt.Errorf("cannot complete hall call that is being processed by another elevator")
		}
	}

	worldview.HallCalls[floor][dir] = resultHallCall
	return nil
}

// CompleteHallCall tries to mark the given hall call as completed.
func (worldview *Worldview) CompleteHallCall(floor int, dir HallCallDirection) error {
	if err := worldview.setHallCall(floor, dir, HallCallStateNone); err != nil {
		return err
	}
	worldview.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: true}
	return nil
}

// NewHallCall tries to create a new order in the system.
func (worldview *Worldview) NewHallCall(floor int, dir HallCallDirection) error {
	worldview.mutex.RLock()

	if worldview.isDisconnected() {
		worldview.mutex.RUnlock()
		return fmt.Errorf("cannot place new hall call when network is disconnected")
	}

	aliveCount := 0
	for _, state := range worldview.ElevatorStates {
		if state.Alive {
			aliveCount++
		}
	}

	localElevator := worldview.ElevatorStates[worldview.LocalID]
	if aliveCount <= 1 && localElevator.IsObstructed {
		worldview.mutex.RUnlock()
		return fmt.Errorf("cannot place new hall call when alone on network and local elevator is obstructed")
	}

	var targetState HallCallState

	worldview.mutex.RUnlock()

	// Alone on network , skip Unconfirmed, go straight to Confirmed
	if aliveCount <= 1 {
		targetState = HallCallStateConfirmed
	} else {
		targetState = HallCallStateUnconfirmed
	}

	if err := worldview.setHallCall(floor, dir, targetState); err != nil {
		return err
	}

	worldview.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
	return nil
}

// ProcessHallCall tries to process the give hall call.
func (worldview *Worldview) ProcessHallCall(floor int, dir HallCallDirection) error {
	return worldview.setHallCall(floor, dir, HallCallStateProcessing)
}

// SetCabCall tries to set the cab call state at given floor.
func (worldview *Worldview) SetCabCall(floor int, state bool) error {
	worldview.mutex.Lock()
	defer worldview.mutex.Unlock()

	if !IsValidFloor(floor, worldview.NumFloors) {
		return fmt.Errorf("%v is not valid floor", floor)
	}

	worldview.ElevatorStates[worldview.LocalID].CabCalls[floor] = state

	worldview.orderUpdateChan <- Order{Type: elevio.Cab, Floor: floor, Completed: !state}

	return nil
}

// SetLocalElevatorStates tries to update state of the local elevator in the worldview.
func (worldview *Worldview) SetLocalElevatorStates(elev *RemoteElevatorState) error {
	worldview.mutex.Lock()
	defer worldview.mutex.Unlock()
	if err := ValidateStateRemote(elev); err != nil {
		return err
	}

	worldview.ElevatorStates[worldview.LocalID] = elev
	return nil
}

// GetRemoteElevatorState returns the local elevator state from the worldview.
func (worldview *Worldview) GetRemoteElevatorState() RemoteElevatorState {
	worldview.mutex.RLock()
	defer worldview.mutex.RUnlock()
	return *worldview.ElevatorStates[worldview.LocalID]
}

// GetAllHallCalls returns a copy of the current hall calls from the worldview.
func (worldview *Worldview) GetAllHallCalls() [][2]HallCallPairState {
	worldview.mutex.RLock()
	defer worldview.mutex.RUnlock()

	result := make([][2]HallCallPairState, len(worldview.HallCalls))
	copy(result, worldview.HallCalls)

	return result
}

// Merge tries to merge incoming Worldview with ours.
// If successful, merges shared state only.
func (worldview *Worldview) Merge(other *Worldview, otherChecksum uint64) error {
	worldview.mutex.Lock()
	defer worldview.mutex.Unlock()

	if other == nil {
		return fmt.Errorf("cannot merge with nil worldview")
	}

	if len(other.HallCalls) != len(worldview.HallCalls) {
		return fmt.Errorf("length of hall calls doesnt match")
	}

	if other.NumFloors != worldview.NumFloors {
		return fmt.Errorf("number of floors doesnt match")
	}

	calculatedChecksum, err := checksum.CalculateChecksum(other)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != otherChecksum {
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	otherLocalState := other.ElevatorStates[other.LocalID]
	if err = ValidateStateRemote(otherLocalState); err != nil {
		return fmt.Errorf("%v's local state is invalid: %w", other.LocalID, err)
	}

	_, weExist := other.ElevatorStates[worldview.LocalID]

	worldview.ElevatorStates[other.LocalID] = otherLocalState

	fetchCabCalls := false

	if !worldview.hasFetchedCabCalls && weExist {
		fetchCabCalls = !slices.Equal(worldview.ElevatorStates[worldview.LocalID].CabCalls, other.ElevatorStates[worldview.LocalID].CabCalls)
	}

	if fetchCabCalls {
		worldview.fetchCabCallsOnReconnect(other)
		worldview.hasFetchedCabCalls = true
	}

	// Merge hall calls
	for floor := range other.HallCalls {
		for dir := range other.HallCalls[floor] {
			otherHallCall := other.HallCalls[floor][dir]
			ourHallCall := worldview.HallCalls[floor][dir]
			dir := HallCallDirection(dir)
			switch otherHallCall.State {
			case HallCallStateNone:
				if ourHallCall.State == HallCallStateProcessing && ourHallCall.AssignedBy == other.LocalID {
					slog.Info("order completed", "by", ourHallCall.AssignedBy, "floor", floor, "dir", dir, "prevState", ourHallCall.State)

					worldview.HallCalls[floor][dir] = HallCallPairState{
						State:      HallCallStateNone,
						AssignedBy: UnassignedID,
						Timestamp:  time.Now().UnixMilli(), // keep timestamp on None to mark "just completed"
					}

					worldview.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: true}
				}
			case HallCallStateUnconfirmed:
				if ourHallCall.State == HallCallStateNone {
					slog.Info("new uncofirmed order", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir)
					worldview.HallCalls[floor][dir].State = HallCallStateUnconfirmed

					worldview.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
				}

				if ourHallCall.State == HallCallStateUnconfirmed {
					slog.Info("order confirmed, promoting to CONFRIMED", "floor", floor, "dir", dir)
					worldview.HallCalls[floor][dir].State = HallCallStateConfirmed
				}
			case HallCallStateConfirmed:
				if ourHallCall.State == HallCallStateNone || ourHallCall.State == HallCallStateUnconfirmed {
					// Peer already promoted,  accept it
					worldview.HallCalls[floor][dir].State = HallCallStateConfirmed

					if ourHallCall.State == HallCallStateNone {
						worldview.orderUpdateChan <- Order{Type: HallDirToButtonType(dir), Floor: floor, Completed: false}
					}
				}

				if ourHallCall.State == HallCallStateProcessing && ourHallCall.AssignedBy == other.LocalID {
					slog.Warn("order has been released", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir)
					worldview.HallCalls[floor][dir] = HallCallPairState{
						State:      HallCallStateConfirmed,
						AssignedBy: UnassignedID,
						Timestamp:  0,
					}
				}

			case HallCallStateProcessing:
				// We can accept if our is 'None' because we might be coming from an initial startup state
				if ourHallCall.State == HallCallStateConfirmed || ourHallCall.State == HallCallStateNone {
					if otherHallCall.AssignedBy == other.LocalID {
						slog.Info("processing order", "by", otherHallCall.AssignedBy, "floor", floor, "dir", dir, "timestamp", otherHallCall.Timestamp)
						worldview.HallCalls[floor][dir].AssignedBy = otherHallCall.AssignedBy
						worldview.HallCalls[floor][dir].State = HallCallStateProcessing
						worldview.HallCalls[floor][dir].Timestamp = otherHallCall.Timestamp
					}
				}
			}
		}
	}

	return nil
}
