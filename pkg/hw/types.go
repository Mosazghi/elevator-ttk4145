package elevio

type (
	MotorDirection int
	ButtonType     int
)

const (
	MDUp   MotorDirection = 1
	MDDown MotorDirection = -1
	MDStop MotorDirection = 0

	HallUp   ButtonType = 0
	HallDown ButtonType = 1
	Cab      ButtonType = 2
)

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

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}
