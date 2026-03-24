package elevio

type (
	// MotorDirection is the commanded elevator travel direction.
	MotorDirection int
	// ButtonType identifies hall up/down and cab buttons.
	ButtonType int
)

const (
	MDUp   MotorDirection = 1
	MDDown MotorDirection = -1
	MDStop MotorDirection = 0

	HallUp   ButtonType = 0
	HallDown ButtonType = 1
	Cab      ButtonType = 2
)

// String returns a readable direction name.
func (md MotorDirection) String() string {
	switch md {
	case MDUp:
		return "up"
	case MDDown:
		return "down"
	case MDStop:
		return "stop"
	default:
		return "Unknown"
	}
}

// ButtonEvent represents one detected button press.
type ButtonEvent struct {
	Floor  int
	Button ButtonType
}
