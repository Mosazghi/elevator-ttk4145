package statesync

import (
	"sync"

	remote "Heisern/pkg/remote-elevator"
)

type HallCallPair struct {
	Up   bool
	Down bool
}

type Worldview struct {
	localID             int
	elevatorStates      map[int]remote.RemoteElevatorState
	hallCalls           map[int]HallCallPair
	syncLocalRemoteChan chan remote.RemoteElevatorState
	localRemoteState    remote.RemoteElevatorState
	numFloors           int
	mtx                 *sync.Mutex
}
