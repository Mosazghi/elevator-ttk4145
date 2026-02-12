package statesync

import (
	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
	"github.com/vmihailenco/msgpack/v5"
)

func BuildWvJson(wv *Worldview) ([]byte, error) {
	checksum, err := checksum.CalculateChecksum(wv)

	if err != nil {
		return nil, err
	}

	msg := Message{Wv: *wv, Checksum: checksum}

	data, err := msgpack.Marshal(msg)

	if err != nil {
		return nil, err
	}

	return data, nil
}
