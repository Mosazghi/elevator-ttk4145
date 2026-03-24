package elevio

// FakeDriver is used tests/simulation without hardware.
type FakeDriver struct{ numFloors int }

// NewElevIoFakeDriver creates a fake driver with the given floor count.
func NewElevIoFakeDriver(numfloors int) *FakeDriver {
	return &FakeDriver{
		numfloors,
	}
}

func (e *FakeDriver) ReadInitialButtons() [4][3]bool {
	var orders [4][3]bool
	return orders
}

func (e *FakeDriver) SetMotorDirection(dir MotorDirection)                   {}
func (e *FakeDriver) SetButtonLamp(button ButtonType, floor int, value bool) {}
func (e *FakeDriver) SetFloorIndicator(floor int)                            {}
func (e *FakeDriver) SetDoorOpenLamp(value bool)                             {}
func (e *FakeDriver) SetStopLamp(value bool)                                 {}
func (e *FakeDriver) GetButton(button ButtonType, floor int) bool            { return false }
func (e *FakeDriver) GetFloor() int                                          { return 0 }
func (e *FakeDriver) GetStop() bool                                          { return false }
func (e *FakeDriver) GetObstruction() bool                                   { return false }
func (e *FakeDriver) PollButtons(receiver chan<- ButtonEvent)                {}
func (e *FakeDriver) PollFloorSensor(receiver chan<- int)                    {}
func (e *FakeDriver) PollStopButton(receiver chan<- bool)                    {}
func (e *FakeDriver) PollObstructionSwitch(receiver chan<- bool)             {}
