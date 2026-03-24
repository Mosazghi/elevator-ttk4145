package controller

// A Penalty is a value used to mark bad behaviors or situations.
//
// These are primarly used during hall call assigment and finding the closest orders.
const (
	PenaltyDoorOpen                  = 30
	PenaltyObstructed                = 20
	PenaltyOppositeMotorDirection    = 10
	PenaltyOppositeHallCallDirection = 5
)
