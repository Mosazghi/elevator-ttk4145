// checksum - utils
package checksum

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// CalculateChecksum computes sha256 checksum of the given data structure.
// The data structure is first marshaled to JSON format.
// Returns the checksum as a byte slice or an error if marshalling fails.
func CalculateChecksum(data any) (uint64, error) {
	byteSlice, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	h := sha256.New()
	h.Write(byteSlice)

	sum := h.Sum(nil)
	if len(sum) < 8 {
		return 0, fmt.Errorf("checksum length is less than 8 bytes")
	}

	var result uint64
	for i := range 8 {
		result = (result << 8) | uint64(sum[i])
	}

	return result, nil
}
