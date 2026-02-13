package statesync

import (
	"encoding/json"

	"github.com/Mosazghi/elevator-ttk4145/shared/checksum"
)

func BuildWvJson(wv *Worldview) ([]byte, error) {
	checksum, err := checksum.CalculateChecksum(wv)

	if err != nil {
		return nil, err
	}

	msg := Message{Wv: *wv, Checksum: checksum}

	jsonData, err := json.Marshal(msg)

	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
