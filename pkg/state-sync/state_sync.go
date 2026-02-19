package statesync

import (
	"sync"

	remoteelevator "github.com/Mosazghi/elevator-ttk4145/pkg/remote-elevator"
)

type HallCallPair struct {
	Up   bool
	Down bool
}

type Worldview struct {
	localID             int
	elevatorStates      map[int]remoteelevator.RemoteElevatorState
	hallCalls           map[int]HallCallPair
	syncLocalRemoteChan chan remoteelevator.RemoteElevatorState
	localRemoteState    remoteelevator.RemoteElevatorState
	numFloors           int
	mtx                 *sync.Mutex
}
