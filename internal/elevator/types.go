package elevator

type (
	Behavior   int
	DoorState  int
	LightState int
)

// Eleavtor behavior states
const (
	BehaviorIdle Behavior = iota
	BehaviorMoving
	BehaviorDoorOpen
)

func (behavior Behavior) String() string {
	switch behavior {
	case BehaviorIdle:
		return "idle"
	case BehaviorMoving:
		return "moving"
	case BehaviorDoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}
