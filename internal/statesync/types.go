package statesync

type (
	HallCallState int
	HallCallDir   int
)

const (
	HallCallStateNone HallCallState = iota
	HallCallStateUnconfirmed
	HallCallStateConfirmed
	HallCallStateProcessing
)

func (hcs HallCallState) String() string {
	switch hcs {
	case HallCallStateNone:
		return "None"
	case HallCallStateUnconfirmed:
		return "Unconfirmed"
	case HallCallStateConfirmed:
		return "Confirmed"
	case HallCallStateProcessing:
		return "Processing"
	default:
		return "Unknown"
	}
}

const (
	HDUp HallCallDir = iota
	HDDown
)

func (hd HallCallDir) String() string {
	switch hd {
	case HDUp:
		return "Up"
	case HDDown:
		return "Down"
	default:
		return "Unknown"
	}
}

type HallCallPairState struct {
	State      HallCallState // the state of the hall call (available, processing, none)
	AssignedBy int           // the id of the elevator that has taken the order
	Timestamp  int64         // the timestamp of when the order was being processed
}

type Order struct {
	Floor     int
	Direction HallCallDir
	Completed bool
}

type Message struct {
	Worldview Worldview
	Checksum  uint64
}
