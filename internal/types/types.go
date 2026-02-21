package types

type (
	Behavior   int
	DoorState  int
	LightState int
)

const (
	BIdle Behavior = iota
	BMoving
	BObstructed
)

const (
	DSClosed DoorState = iota
	DSOpen
)

const (
	LSOff LightState = iota
	LSOn
)

func (b Behavior) String() string {
	switch b {
	case BIdle:
		return "IDLE"
	case BMoving:
		return "MOVING"
	case BObstructed:
		return "OBSTRUCTED"
	}
	return "UNKNOWN"
}

func (d DoorState) String() string {
	switch d {
	case DSClosed:
		return "CLOSED"
	case DSOpen:
		return "OPEN"
	}
	return "UNKNOWN"
}

func (l LightState) String() string {
	switch l {
	case LSOff:
		return "Off"
	case LSOn:
		return "On"
	}
	return "UNKNOWN"
}
