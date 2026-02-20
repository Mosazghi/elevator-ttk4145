package statesync

type (
	HallCallState int
	HallCallDir   int
)

const (
	HSNone HallCallState = iota
	HSAvailable
	HSProcessing
)

const (
	HDDown HallCallDir = iota
	HDUp
)

type HallCallPairState struct {
	State       HallCallState // the state of the hall call (available, processing, none)
	By          int           // the id of the elevator that has taken the order
	ConfirmedBy []int         // the ids of the elevators that have (seen) confirmed the order
	Timestamp   int64         // the timestamp of when the order was being processed
}
