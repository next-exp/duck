package duck

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStatistics_JSONSerialization(t *testing.T) {
	stats := RunStatistics{
		Host:   "testhost",
		Events: 1000,
		Bytes:  50000,
		Errors: 5,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "testhost", jsonMap["host"])
	assert.Equal(t, float64(1000), jsonMap["events"])
	assert.Equal(t, float64(50000), jsonMap["bytes"])
	assert.Equal(t, float64(5), jsonMap["errors"])
}

func TestRunStatistics_JSONDeserialization(t *testing.T) {
	jsonData := `{"host":"server1","events":2500,"bytes":125000,"errors":10}`

	var stats RunStatistics
	err := json.Unmarshal([]byte(jsonData), &stats)
	require.NoError(t, err)

	assert.Equal(t, "server1", stats.Host)
	assert.Equal(t, int64(2500), stats.Events)
	assert.Equal(t, int64(125000), stats.Bytes)
	assert.Equal(t, int64(10), stats.Errors)
}

func TestRunStatistics_JSONRoundtrip(t *testing.T) {
	original := RunStatistics{
		Host:   "roundtrip-test",
		Events: 999999,
		Bytes:  888888,
		Errors: 777,
	}

	// Marshal
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var restored RunStatistics
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original, restored)
}

func TestRunStatistics_ZeroValues(t *testing.T) {
	stats := RunStatistics{}

	// Marshal to JSON
	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)

	// Verify JSON structure with zero values
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "", jsonMap["host"])
	assert.Equal(t, float64(0), jsonMap["events"])
	assert.Equal(t, float64(0), jsonMap["bytes"])
	assert.Equal(t, float64(0), jsonMap["errors"])
}

func TestRunStatistics_LargeValues(t *testing.T) {
	// Test with large int64 values
	stats := RunStatistics{
		Host:   "large-values-test",
		Events: math.MaxInt64 / 2,
		Bytes:  math.MaxInt64 / 2,
		Errors: math.MaxInt32,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)

	// Unmarshal back
	var restored RunStatistics
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, stats.Host, restored.Host)
	assert.Equal(t, stats.Events, restored.Events)
	assert.Equal(t, stats.Bytes, restored.Bytes)
	assert.Equal(t, stats.Errors, restored.Errors)
}

func TestRunStatistics_NegativeValues(t *testing.T) {
	// While negative values may not make semantic sense,
	// the struct should handle them correctly
	stats := RunStatistics{
		Host:   "negative-test",
		Events: -100,
		Bytes:  -1000,
		Errors: -5,
	}

	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)

	var restored RunStatistics
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	assert.Equal(t, int64(-100), restored.Events)
	assert.Equal(t, int64(-1000), restored.Bytes)
	assert.Equal(t, int64(-5), restored.Errors)
}
