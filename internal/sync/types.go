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
	HDUp = iota
	HDDown
	HDNone
)

type HallCallPairState struct {
	State HallCallState
	By    int
}
