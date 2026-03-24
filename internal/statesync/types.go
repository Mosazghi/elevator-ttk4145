package statesync

import elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"

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

// HallCallPairState represents the state of a hall call.
type HallCallPairState struct {
	// State represents the current state of the hall call (None, Unconfirmed, Confirmed, Processing).
	State HallCallState
	// AssignedBy is the ID of the elevator that has accepted to serve this hall call. Only relevant when State is Processing.
	AssignedBy int
	// Timestamp is the time when the hall call was accepted by an elevator. Only relevant when State is Processing.
	Timestamp int64
}

type Order struct {
	Type      elevio.ButtonType
	Floor     int
	Completed bool
}

// Message is whats sent over the network.
type Message struct {
	Worldview Worldview
	Checksum  uint64
}
