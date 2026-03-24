package controller

// A ControllerTriggerSrc is a single value representing where a
// trigger is arrived from.
type ControllerTriggerSrc int

const (
	ControllerTriggerSrcArrivalFloor ControllerTriggerSrc = iota
	ControllerTriggerSrcOrderUpdate
)
