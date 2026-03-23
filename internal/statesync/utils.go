package statesync

import (
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
	"github.com/vmihailenco/msgpack/v5"
)

// BuildWvJSON constructs a json message from a Worldview
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

func HallDirToButtonType(dir HallCallDir) elevio.ButtonType {
	if dir == HDDown {
		return elevio.HallDown
	}
	return elevio.HallUp
}
