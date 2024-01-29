package duck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateType_String_AllValues(t *testing.T) {
	tests := []struct {
		name     string
		state    StateType
		expected string
	}{
		{
			name:     "INITIALIZED state",
			state:    INITIALIZED,
			expected: "INITIALIZED",
		},
		{
			name:     "RUNNING state",
			state:    RUNNING,
			expected: "RUNNING",
		},
		{
			name:     "STARTING state",
			state:    STARTING,
			expected: "STARTING",
		},
		{
			name:     "STOPPING state",
			state:    STOPPING,
			expected: "STOPPING",
		},
		{
			name:     "PINGING state",
			state:    PINGING,
			expected: "PINGING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStateType_String_InvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		state    StateType
		expected string
	}{
		{
			name:     "negative value",
			state:    StateType(-1),
			expected: "StateType(-1)",
		},
		{
			name:     "value beyond last state",
			state:    StateType(5),
			expected: "StateType(5)",
		},
		{
			name:     "large invalid value",
			state:    StateType(100),
			expected: "StateType(100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStateType_String_BoundaryCases(t *testing.T) {
	tests := []struct {
		name     string
		state    StateType
		expected string
	}{
		{
			name:     "first valid state (0)",
			state:    StateType(0),
			expected: "INITIALIZED",
		},
		{
			name:     "last valid state (4)",
			state:    StateType(4),
			expected: "PINGING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
