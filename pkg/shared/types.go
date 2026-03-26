// Package shared contains shared types and const used thoughout the program
package shared

// Empty represent an empty type. Its main purpose is to clearify that nothing can be
// expected when using this type.
//
// This is commonly used for channels where returning values are not needed.
type Empty struct{}

const (
	// UndefinedFloor is the default number used during initialization from
	// an unkown position.
	UndefinedFloor = -1
	// ID for hall call order than has not been assigned to any node
	UnassignedID = -1
)
