package duck

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnToken tests JWT token generation
func TestConnToken(t *testing.T) {
	user := "testuser"
	secret := "testsecret"

	token := ConnToken(user, 0, secret)

	assert.NotEmpty(t, token)

	// Parse the token to verify it's valid JWT
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user, claims["sub"])
}

// TestConnToken_WithExpiration tests token generation with expiration
func TestConnToken_WithExpiration(t *testing.T) {
	user := "testuser"
	secret := "testsecret"
	exp := time.Now().Add(1 * time.Hour).Unix()

	token := ConnToken(user, exp, secret)

	assert.NotEmpty(t, token)

	// Parse and verify
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	// Verify expiration claim
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, float64(exp), claims["exp"])
}

// TestConnToken_DifferentUsers tests tokens for different users
func TestConnToken_DifferentUsers(t *testing.T) {
	secret := "testsecret"
	users := []string{"user1", "user2", "user3"}

	tokens := make(map[string]string)

	for _, user := range users {
		token := ConnToken(user, 0, secret)
		tokens[user] = token

		// Verify each token
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		require.True(t, ok)
		assert.Equal(t, user, claims["sub"])
	}

	// Ensure all tokens are different
	assert.NotEqual(t, tokens["user1"], tokens["user2"])
	assert.NotEqual(t, tokens["user2"], tokens["user3"])
	assert.NotEqual(t, tokens["user1"], tokens["user3"])
}

// TestConnToken_SigningMethod tests that HS256 is used
func TestConnToken_SigningMethod(t *testing.T) {
	user := "testuser"
	secret := "testsecret"

	token := ConnToken(user, 0, secret)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		assert.True(t, ok, "Expected HMAC signing method")
		return []byte(secret), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "HS256", parsedToken.Header["alg"])
}

// TestMetrics_JSONSerialization tests Metrics JSON marshalling
func TestMetrics_JSONSerialization(t *testing.T) {
	metrics := Metrics{
		EventCounter:    500,
		ByteCounter:     2048,
		CurrentTrgRate:  15.3,
		CurrentDataRate: 2048.7,
		AvgTrgRate:      14.1,
		AvgDataRate:     1900.2,
	}

	data, err := json.Marshal(metrics)
	require.NoError(t, err)

	var decoded Metrics
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, metrics.EventCounter, decoded.EventCounter)
	assert.Equal(t, metrics.ByteCounter, decoded.ByteCounter)
	assert.Equal(t, metrics.CurrentTrgRate, decoded.CurrentTrgRate)
	assert.Equal(t, metrics.CurrentDataRate, decoded.CurrentDataRate)
	assert.Equal(t, metrics.AvgTrgRate, decoded.AvgTrgRate)
	assert.Equal(t, metrics.AvgDataRate, decoded.AvgDataRate)
}

// TestState_JSONSerialization tests State JSON marshalling
func TestState_JSONSerialization(t *testing.T) {
	states := []string{"INITIALIZED", "RUNNING", "STARTING", "STOPPING", "PINGING"}

	for _, s := range states {
		state := State{State: s}

		data, err := json.Marshal(state)
		require.NoError(t, err)

		var decoded State
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, s, decoded.State)
	}
}

// TestOutputFile_JSONSerialization tests OutputFile JSON marshalling
func TestOutputFile_JSONSerialization(t *testing.T) {
	testCases := []struct {
		server string
		subrun int
	}{
		{"gdc1", 0},
		{"gdc2", 5},
		{"ldc1", 100},
		{"server-123", 999},
	}

	for _, tc := range testCases {
		outputFile := OutputFile{
			Server: tc.server,
			Subrun: tc.subrun,
		}

		data, err := json.Marshal(outputFile)
		require.NoError(t, err)

		var decoded OutputFile
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, tc.server, decoded.Server)
		assert.Equal(t, tc.subrun, decoded.Subrun)
	}
}

// TestCentrifugeTopic tests the constant
func TestCentrifugeTopic(t *testing.T) {
	assert.Equal(t, "duck", CENTRIFUGE_TOPIC)
}

// TestMetrics_EdgeCases tests Metrics JSON serialization with various edge cases
func TestMetrics_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  Metrics
		checkMsg bool // whether to verify field names in JSON
	}{
		{
			name: "Normal values",
			metrics: Metrics{
				EventCounter:    500,
				ByteCounter:     2048,
				CurrentTrgRate:  15.3,
				CurrentDataRate: 2048.7,
				AvgTrgRate:      14.1,
				AvgDataRate:     1900.2,
			},
			checkMsg: true,
		},
		{
			name:     "Zero values",
			metrics:  Metrics{},
			checkMsg: false,
		},
		{
			name: "Negative values",
			metrics: Metrics{
				EventCounter:    -1,
				ByteCounter:     -100,
				CurrentTrgRate:  -5.5,
				CurrentDataRate: -1024.0,
				AvgTrgRate:      -10.0,
				AvgDataRate:     -2000.0,
			},
			checkMsg: false,
		},
		{
			name: "Large values",
			metrics: Metrics{
				EventCounter:    1000000000,
				ByteCounter:     9999999999,
				CurrentTrgRate:  999999.999,
				CurrentDataRate: 9999999.999,
				AvgTrgRate:      888888.888,
				AvgDataRate:     8888888.888,
			},
			checkMsg: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should be serializable
			data, err := json.Marshal(tc.metrics)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify field names for normal case
			if tc.checkMsg {
				dataStr := string(data)
				assert.Contains(t, dataStr, "EventCounter")
				assert.Contains(t, dataStr, "ByteCounter")
			}

			// Should be deserializable
			var decoded Metrics
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tc.metrics.EventCounter, decoded.EventCounter)
			assert.Equal(t, tc.metrics.ByteCounter, decoded.ByteCounter)

			// Use delta for float comparisons
			if tc.name == "Large values" {
				assert.InDelta(t, tc.metrics.CurrentTrgRate, decoded.CurrentTrgRate, 0.001)
				assert.InDelta(t, tc.metrics.CurrentDataRate, decoded.CurrentDataRate, 0.001)
			}
		})
	}
}

// TestConnToken_EmptyUser tests token generation with empty user
func TestConnToken_EmptyUser(t *testing.T) {
	secret := "testsecret"

	token := ConnToken("", 0, secret)

	assert.NotEmpty(t, token)

	// Parse and verify
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	require.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "", claims["sub"])
}

// TestConnToken_LongSecret tests token generation with long secret
func TestConnToken_LongSecret(t *testing.T) {
	user := "testuser"
	secret := "this-is-a-very-long-secret-key-for-testing-purposes-with-many-characters"

	token := ConnToken(user, 0, secret)

	assert.NotEmpty(t, token)

	// Verify it can be parsed with the same secret
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)
}

// TestConnToken_ExpiredToken tests token with past expiration
func TestConnToken_ExpiredToken(t *testing.T) {
	user := "testuser"
	secret := "testsecret"
	exp := time.Now().Add(-1 * time.Hour).Unix() // Expired 1 hour ago

	token := ConnToken(user, exp, secret)

	assert.NotEmpty(t, token)

	// Parse the token (it will be invalid due to expiration)
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	// Token parsing succeeds but Valid will be false
	require.Error(t, err)
	assert.False(t, parsedToken.Valid)
}

// TestMetrics_JSONFieldNames tests that JSON field names match expected format
func TestMetrics_JSONFieldNames(t *testing.T) {
	metrics := Metrics{
		EventCounter: 100,
		ByteCounter:  200,
	}

	data, err := json.Marshal(metrics)
	require.NoError(t, err)

	// Verify field names are PascalCase (Go default)
	dataStr := string(data)
	assert.Contains(t, dataStr, "EventCounter")
	assert.Contains(t, dataStr, "ByteCounter")
}

// TestMessage_AllMessageTypes tests all message types
func TestMessage_AllMessageTypes(t *testing.T) {
	messageTypes := []MessageType{
		MessageError,
		MessageInfo,
		MessageDebug,
		MessageMetric,
		MessageState,
		MessageFile,
		MessageSummary,
	}

	for _, msgType := range messageTypes {
		msg := Message{
			Timestamp:     time.Now(),
			Host:          "testhost",
			Type:          msgType,
			Value:         "test",
			StopProcesses: false,
			RunNumber:     1,
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded Message
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msgType, decoded.Type)
	}
}

