package elevio

import "time"

const (
	// pollRate is the sampling period used by input polling loops.
	pollRate = 20 * time.Millisecond
	// debounceTime is the minimum stable duration before stop-button changes emit.
	debounceTime = 100 * time.Millisecond
)
