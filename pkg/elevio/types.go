package elevio

type (
	// MotorDirection is the commanded elevator travel direction.
	MotorDirection int
	// ButtonType identifies hall up/down and cab buttons.
	ButtonType int
)

const (
	MotorDirectionUp   MotorDirection = 1
	MotorDirectionDown MotorDirection = -1
	MotorDirectionStop MotorDirection = 0

	HallUp   ButtonType = 0
	HallDown ButtonType = 1
	Cab      ButtonType = 2
)

// String returns a readable direction name.
func (motorDirection MotorDirection) String() string {
	switch motorDirection {
	case MotorDirectionUp:
		return "up"
	case MotorDirectionDown:
		return "down"
	case MotorDirectionStop:
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
