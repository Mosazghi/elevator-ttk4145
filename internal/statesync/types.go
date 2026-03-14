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

const (
	HDUp HallCallDir = iota
	HDDown
)

type HallCallPairState struct {
	State     HallCallState // the state of the hall call (available, processing, none)
	By        int           // the id of the elevator that has taken the order
	Timestamp int64         // the timestamp of when the order was being processed
}
