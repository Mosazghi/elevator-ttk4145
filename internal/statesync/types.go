package statesync

import elevio "github.com/Mosazghi/elevator-ttk4145/pkg/hw"

type (
	HallCallState     int
	HallCallDirection int
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
	HallCallDirectionUp HallCallDirection = iota
	HallCallDirectionDown
)

func (hallCallDirection HallCallDirection) String() string {
	switch hallCallDirection {
	case HallCallDirectionUp:
		return "Up"
	case HallCallDirectionDown:
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
	Type      elevio.ButtonType
	Floor     int
	Completed bool
}

type Message struct {
	Worldview Worldview
	Checksum  uint64
}
