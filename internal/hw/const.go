package elevio

import "time"

const (
	pollRate     = 20 * time.Millisecond
	debounceTime = 100 * time.Millisecond
)
