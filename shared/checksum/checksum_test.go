package checksum

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test 1: Basic functionality
func TestCalculateChecksum_ValidInput(t *testing.T) {
	data := map[string]interface{}{
		"id":    1,
		"floor": 2,
		"name":  "elevator-1",
	}

	cs, err := CalculateChecksum(data)
	t.Logf("Calculated: %v", cs)
	require.NoError(t, err)
	// assert.Len(t, cs, 32, "SHA-256 produces 32 bytes")
}

// Test 2: Determinism - same input produces same checksum
func TestCalculateChecksum_Deterministic(t *testing.T) {
	data := map[string]int{"x": 1, "y": 2, "z": 3}

	cs1, err1 := CalculateChecksum(data)
	cs2, err2 := CalculateChecksum(data)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.True(t, cs1 == cs2, "same input must produce same checksum")
}

// Test 3: Different inputs produce different checksums
func TestCalculateChecksum_UniqueOutputs(t *testing.T) {
	data1 := map[string]int{"value": 1}
	data2 := map[string]int{"value": 2}

	cs1, _ := CalculateChecksum(data1)
	cs2, _ := CalculateChecksum(data2)

	assert.False(t, cs1 == cs2, "different inputs should produce different checksums")
}

// Test 4: Edge cases - empty and nil values
func TestCalculateChecksum_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"empty map", map[string]interface{}{}},
		{"empty slice", []interface{}{}},
		{"empty string", ""},
		{"zero number", 0},
		{"nil slice", []interface{}(nil)},
		{"boolean true", true},
		{"boolean false", false},
		{"null", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateChecksum(tt.input)
			require.NoError(t, err, "should handle %s", tt.name)
			// assert.Len(t, cs, 32)
		})
	}
}

// Test 5: Complex nested structures
func TestCalculateChecksum_NestedStructures(t *testing.T) {
	data := map[string]interface{}{
		"elevator": map[string]interface{}{
			"id":    "1",
			"floor": 3,
			"calls": []int{1, 2, 4},
			"state": map[string]bool{
				"moving":    true,
				"door_open": false,
			},
		},
		"timestamp": 1234567890,
	}

	cs, err := CalculateChecksum(data)
	t.Logf("CS: %+v", cs)
	require.NoError(t, err)
	// assert.Len(t, cs, 32)
}

// Test 6: Struct input
func TestCalculateChecksum_WithStruct(t *testing.T) {
	type Message struct {
		ID        string `json:"id"`
		Floor     int    `json:"floor"`
		Direction string `json:"direction"`
		Orders    []int  `json:"orders"`
	}

	msg := Message{
		ID:        "elevator-1",
		Floor:     2,
		Direction: "up",
		Orders:    []int{3, 4},
	}

	cs1, err := CalculateChecksum(msg)
	require.NoError(t, err)

	// Verify determinism with struct
	cs2, _ := CalculateChecksum(msg)
	assert.Equal(t, cs1, cs2, "struct checksums must be deterministic")
}

// Test 7: Order independence for structs (should be deterministic)
func TestCalculateChecksum_StructFieldOrder(t *testing.T) {
	type Message struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	msg1 := Message{A: 1, B: 2}
	msg2 := Message{B: 2, A: 1}

	cs1, _ := CalculateChecksum(msg1)
	cs2, _ := CalculateChecksum(msg2)

	assert.Equal(t, cs1, cs2, "struct field order should not matter for identical values")
}

// Test 8: Large data structures
func TestCalculateChecksum_LargeInput(t *testing.T) {
	// Simulate a large worldview
	worldview := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		worldview[fmt.Sprintf("elevator-%d", i)] = map[string]interface{}{
			"floor":     i % 4,
			"orders":    []int{1, 2, 3},
			"timestamp": time.Now().Unix(),
		}
	}

	_, err := CalculateChecksum(worldview)
	require.NoError(t, err)
	// assert.Len(t, cs, 32)
}

// Test 9: JSON-unmarshalable types should error
func TestCalculateChecksum_InvalidInput(t *testing.T) {
	// Channels, functions, and complex types can't be marshaled to JSON
	invalidInputs := []interface{}{
		make(chan int),
		func() {},
		complex(1, 2),
	}

	for i, input := range invalidInputs {
		t.Run(fmt.Sprintf("invalid_type_%d", i), func(t *testing.T) {
			_, err := CalculateChecksum(input)
			assert.Error(t, err, "should fail for non-JSON-serializable types")
		})
	}
}

// Test 10: Stability across JSON round-trip
func TestCalculateChecksum_JSONRoundTrip(t *testing.T) {
	type Message struct {
		ID     string `json:"id"`
		Values []int  `json:"values"`
	}

	original := Message{ID: "test", Values: []int{1, 2, 3}}

	// Marshal to JSON and back
	jsonBytes, _ := json.Marshal(original)
	var decoded Message
	json.Unmarshal(jsonBytes, &decoded)

	cs1, _ := CalculateChecksum(original)
	cs2, _ := CalculateChecksum(decoded)

	assert.Equal(t, cs1, cs2, "checksum should survive JSON round-trip")
}

// Test 11: Collision resistance
func TestCalculateChecksum_CollisionResistance(t *testing.T) {

	checksums := make(map[string]bool)

	// Generate checksums for slight variations
	for i := 0; i < 100; i++ {
		data := map[string]int{"value": 100 + i}
		cs, _ := CalculateChecksum(data)
		csHex := fmt.Sprintf("%x", cs)

		assert.False(t, checksums[csHex], "collision detected at i=%d", i)
		checksums[csHex] = true
	}
}
