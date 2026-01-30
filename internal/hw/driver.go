package elevio

import (
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

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
	GetTotalFloors() int
	GetObstruction() bool
	PollButtons(receiver chan<- ButtonEvent)
	PollFloorSensor(receiver chan<- int)
	PollStopButton(receiver chan<- bool)
	PollObstructionSwitch(receiver chan<- bool)
}

// ElevIoDriver is a struct that implements the ElevatorDriver interface
type ElevIoDriver struct {
	numFloors int
	mtx       sync.Mutex
	conn      net.Conn
}

type MotorDirection int

const (
	Up   MotorDirection = 1
	Down MotorDirection = -1
	Stop MotorDirection = 0
)

type ButtonType int

const (
	HallUp   ButtonType = 0
	HallDown ButtonType = 1
	Cab      ButtonType = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

func NewElevIoDriver(addr string, numFloors int) *ElevIoDriver {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	return &ElevIoDriver{
		numFloors: numFloors,
		mtx:       sync.Mutex{},
		conn:      conn,
	}
}

func (e *ElevIoDriver) GetTotalFloors() int {
	return e.numFloors
}

func (e *ElevIoDriver) ReadInitialButtons() [4][3]bool {
	var orders [4][3]bool
	for f := range orders {
		for b := range orders[f] {
			if e.GetButton(ButtonType(b), f) {
				orders[f][b] = true
			}
		}
	}
	return orders
}

func (e *ElevIoDriver) SetMotorDirection(dir MotorDirection) {
	e.write([4]byte{1, byte(dir), 0, 0})
}

func (e *ElevIoDriver) SetButtonLamp(button ButtonType, floor int, value bool) {
	e.write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func (e *ElevIoDriver) SetFloorIndicator(floor int) {
	e.write([4]byte{3, byte(floor), 0, 0})
}

func (e *ElevIoDriver) SetDoorOpenLamp(value bool) {
	e.write([4]byte{4, toByte(value), 0, 0})
}

func (e *ElevIoDriver) SetStopLamp(value bool) {
	e.write([4]byte{5, toByte(value), 0, 0})
}

func (e *ElevIoDriver) PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, e.numFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < e.numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := e.GetButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func (e *ElevIoDriver) PollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := e.GetFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func (e *ElevIoDriver) PollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := e.GetStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func (e *ElevIoDriver) PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := e.GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func (e *ElevIoDriver) GetButton(button ButtonType, floor int) bool {
	a := e.read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

func (e *ElevIoDriver) GetFloor() int {
	a := e.read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func (e *ElevIoDriver) GetStop() bool {
	a := e.read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func (e *ElevIoDriver) GetObstruction() bool {
	a := e.read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

func (e *ElevIoDriver) read(in [4]byte) [4]byte {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	_, err := e.conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = e.conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func (e *ElevIoDriver) write(in [4]byte) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	_, err := e.conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	b := false
	if a != 0 {
		b = true
	}
	return b
}
