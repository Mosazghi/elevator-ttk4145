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
	HDDown = iota
	HDUp
	HDNone
)

type HallCallPairState struct {
	State       HallCallState
	By          int
	ConfirmedBy []int
}
