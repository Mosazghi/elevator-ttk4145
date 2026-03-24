package statesync

import (
	"github.com/Mosazghi/elevator-ttk4145/pkg/checksum"
	elevio "github.com/Mosazghi/elevator-ttk4145/pkg/elevio"
	"github.com/vmihailenco/msgpack/v5"
)

// BuildWvJSON constructs a json message from a Worldview.
func BuildWvJSON(wv *Worldview) ([]byte, error) {
	checksum, err := checksum.CalculateChecksum(wv)
	if err != nil {
		return nil, err
	}

	msg := Message{Worldview: *wv, Checksum: checksum}

	data, err := msgpack.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// HallDirToButtonType returns the button type based on direction.
func HallDirToButtonType(dir HallCallDirection) elevio.ButtonType {
	if dir == HallCallDirectionDown {
		return elevio.HallDown
	}
	return elevio.HallUp
}
