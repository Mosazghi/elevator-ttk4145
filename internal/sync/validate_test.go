package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStateWv(t *testing.T) {
	wv := &Worldview{}
	assert.True(t, ValidateStateWv(wv), "Expected invalid Worldview state")
}

func TestValidateStateRemote(t *testing.T) {
	state := &RemoteElevatorState{}
	assert.False(t, ValidateStateRemote(state), "Expected invalid RemoteElevatorState")
}
