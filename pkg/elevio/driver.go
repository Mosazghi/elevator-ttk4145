// Package elevio provides the ElevatorDriver interface and a TCP driver implementation for the elevator server/simulator.
package elevio

import (
	"net"
	"sync"
	"time"
)

// ElevatorDriver abstracts hardware I/O for real and fake elevator backends.
type ElevatorDriver interface {
	ReadInitialButtons() [4][3]bool
	SetMotorDirection(dir MotorDirection)
	SetButtonLamp(button ButtonType, floor int, value bool)
	SetFloorIndicator(floor int)
	SetDoorOpenLamp(value bool)
	SetStopLamp(value bool)
	GetButton(button ButtonType, floor int) bool
	GetFloor() int
	GetStop() bool
	GetObstruction() bool
	PollButtons(receiver chan<- ButtonEvent)
	PollFloorSensor(receiver chan<- int)
	PollStopButton(receiver chan<- bool)
	PollObstructionSwitch(receiver chan<- bool)
}

// ElevIoDriver talks to the simulator / elevator server over TCP.
type ElevIoDriver struct {
	numFloors  int
	mutex      sync.Mutex
	connection net.Conn
}

// NewElevIoDriver connects to the elevator server at addr.
func NewElevIoDriver(addr string, numFloors int) *ElevIoDriver {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	return &ElevIoDriver{
		numFloors:  numFloors,
		mutex:      sync.Mutex{},
		connection: conn,
	}
}

// ReadInitialButtons snapshots all currently pressed hall/cab buttons.
func (eid *ElevIoDriver) ReadInitialButtons() [4][3]bool {
	var orders [4][3]bool
	for f := range orders {
		for b := range orders[f] {
			if eid.GetButton(ButtonType(b), f) {
				orders[f][b] = true
			}
		}
	}
	return orders
}

// SetMotorDirection sends motor direction command to hardware.
func (eid *ElevIoDriver) SetMotorDirection(dir MotorDirection) {
	eid.write([4]byte{1, byte(dir), 0, 0})
}

// SetButtonLamp toggles a hall/cab button lamp.
func (eid *ElevIoDriver) SetButtonLamp(button ButtonType, floor int, value bool) {
	eid.write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

// SetFloorIndicator updates the floor indicator lamp.
func (eid *ElevIoDriver) SetFloorIndicator(floor int) {
	eid.write([4]byte{3, byte(floor), 0, 0})
}

// SetDoorOpenLamp toggles the door-open lamp.
func (eid *ElevIoDriver) SetDoorOpenLamp(value bool) {
	eid.write([4]byte{4, toByte(value), 0, 0})
}

// SetStopLamp toggles the stop lamp.
func (eid *ElevIoDriver) SetStopLamp(value bool) {
	eid.write([4]byte{5, toByte(value), 0, 0})
}

// PollButtons emits press edges (not release edges) for all buttons.
func (eid *ElevIoDriver) PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, eid.numFloors)
	for {
		time.Sleep(pollRate)
		for f := 0; f < eid.numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := eid.GetButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

// PollFloorSensor emits floor changes while the elevator is aligned at a floor.
func (eid *ElevIoDriver) PollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(pollRate)
		v := eid.GetFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

// PollStopButton emits debounced stop-button state changes.
func (eid *ElevIoDriver) PollStopButton(receiver chan<- bool) {
	stable := false
	candidate := stable
	candidateSince := time.Now()

	for {
		time.Sleep(pollRate)
		raw := eid.GetStop()

		if raw != candidate {
			candidate = raw
			candidateSince = time.Now()
		}

		if candidate != stable && time.Since(candidateSince) >= debounceTime {
			stable = candidate
			receiver <- stable
		}
	}
}

// PollObstructionSwitch emits obstruction state transitions.
func (eid *ElevIoDriver) PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(pollRate)
		v := eid.GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

// GetButton reads current state for one button at one floor.
func (eid *ElevIoDriver) GetButton(button ButtonType, floor int) bool {
	a := eid.read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

// GetFloor returns current floor, or -1 between floors.
func (eid *ElevIoDriver) GetFloor() int {
	a := eid.read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

// GetStop reads the stop button state.
func (eid *ElevIoDriver) GetStop() bool {
	a := eid.read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

// GetObstruction reads the obstruction switch state.
func (eid *ElevIoDriver) GetObstruction() bool {
	a := eid.read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

// read performs a request/response transaction with serialized access.
func (eid *ElevIoDriver) read(in [4]byte) [4]byte {
	eid.mutex.Lock()
	defer eid.mutex.Unlock()

	_, err := eid.connection.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = eid.connection.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

// write sends a command-only message with serialized access.
func (eid *ElevIoDriver) write(in [4]byte) {
	eid.mutex.Lock()
	defer eid.mutex.Unlock()

	_, err := eid.connection.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

// toByte encodes bool as 0/1 for the wire protocol.
func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

// toBool decodes 0/1 wire values to bool.
func toBool(a byte) bool {
	b := false
	if a != 0 {
		b = true
	}
	return b
}
