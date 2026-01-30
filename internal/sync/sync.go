package sync

import (
	"sync"
)

type HallCallPair struct {
	Up   bool
	Down bool
}

type Worldview struct {
	localID             int
	elevatorStates      map[int]RemoteElevatorState
	hallCalls           map[int]HallCallPair
	syncLocalRemoteChan chan RemoteElevatorState
	localRemoteState    RemoteElevatorState
	numFloors           int
	mtx                 *sync.Mutex
}
