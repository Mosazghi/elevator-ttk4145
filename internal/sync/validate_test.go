package statesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStateWv(t *testing.T) {
	t.Skip()
	wv := &Worldview{}
	assert.False(t, ValidateStateWv(wv), "Expected invalid Worldview state")
}

func TestValidateStateRemote(t *testing.T) {
	t.Skip()
	state := &RemoteElevatorState{}
	assert.False(t, ValidateStateRemote(state), "Expected invalid RemoteElevatorState")
}
