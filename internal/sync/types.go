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
)

type HallCallPairState struct {
	State HallCallState
	By    int
}
