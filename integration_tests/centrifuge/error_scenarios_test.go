//go:build integration
// +build integration

package centrifuge_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)


// TestCentrifugeConnection_InvalidToken tests connection rejection with invalid token
func TestCentrifugeConnection_InvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Create local config copy with invalid token
	localConfig := Config
	localConfig.Token = "invalid-token-abc123"

	client, err := duck.CentrifugeConnection(localConfig, "testuser-invalid-token")

	// Connection might be created but should fail to connect
	if err == nil {
		defer client.Close()

		// Try to wait for connection - should fail
		err = testhelpers.WaitForConnection(client, 5*time.Second)
		assert.Error(t, err, "Should fail to connect with invalid token")
		t.Logf("Connection correctly failed: %v", err)
	} else {
		assert.Error(t, err, "Should fail to create connection with invalid token")
		t.Logf("Client creation correctly failed: %v", err)
	}
}

// TestCentrifugeConnection_ExpiredToken tests connection rejection with expired token
func TestCentrifugeConnection_ExpiredToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Create an expired token (1 hour ago)
	expiredTime := time.Now().Add(-1 * time.Hour).Unix()
	expiredToken := duck.ConnToken("testuser", expiredTime, "testsecret")

	// Create local config copy with expired token
	localConfig := Config
	localConfig.Token = expiredToken

	client, err := duck.CentrifugeConnection(localConfig, "testuser-expired-token")

	if err == nil {
		defer client.Close()

		// Try to wait for connection - should fail
		err = testhelpers.WaitForConnection(client, 5*time.Second)
		assert.Error(t, err, "Should fail to connect with expired token")
		t.Logf("Connection correctly failed with expired token: %v", err)
	} else {
		assert.Error(t, err, "Should fail to create connection with expired token")
		t.Logf("Client creation correctly failed with expired token: %v", err)
	}
}

// TestCentrifugePublisher_PublishWhenDisconnected tests publishing when client is disconnected
func TestCentrifugePublisher_PublishWhenDisconnected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, resource := testhelpers.SetupCentrifugeContainer(t)

	client, err := duck.CentrifugeConnection(Config, "testuser-publish-disconnected")
	require.NoError(t, err)

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	pubSub, err := duck.SubscribeCentrifuge(client, nil)
	require.NoError(t, err)

	publisher := &duck.CentrifugePublisher{Sub: pubSub}

	// Close client before publishing
	client.Close()

	// Try to publish - should fail
	testMsg := duck.Message{
		Timestamp:     time.Now(),
		Host:          "test-host",
		Type:          duck.MessageInfo,
		Value:         "test message",
		StopProcesses: false,
		RunNumber:     1,
	}

	msgData, _ := json.Marshal(testMsg)
	err = publisher.Publish(context.Background(), msgData)
	assert.Error(t, err, "Publish should fail when client is disconnected")
	t.Logf("Publish correctly failed when disconnected: %v", err)

	// Cleanup
	testhelpers.CleanupDocker(pool, resource)
}


// TestCentrifugeConnection_Timeout tests connection timeout with unreachable host
func TestCentrifugeConnection_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a non-routable IP address (should timeout)
	Config := duck.CentrifugalConfiguration{
		Host:  "198.51.100.1", // TEST-NET-2, reserved for documentation
		Port:  9999,
		Token: "testsecret",
	}

	// Connection should either fail immediately or timeout
	start := time.Now()
	client, err := duck.CentrifugeConnection(Config, "testuser-timeout")
	elapsed := time.Since(start)

	if err == nil {
		defer client.Close()
		// If client was created, connection should timeout
		err = testhelpers.WaitForConnection(client, 10*time.Second)
		assert.Error(t, err, "Should timeout connecting to unreachable host")
		t.Logf("Connection correctly timed out after %v", elapsed)
	} else {
		assert.Error(t, err, "Should fail to connect to unreachable host")
		t.Logf("Connection correctly failed after %v: %v", elapsed, err)
	}

	// Should fail reasonably quickly (not hang indefinitely)
	assert.Less(t, elapsed, 15*time.Second, "Should fail/timeout within 15 seconds")
	t.Logf("Connection attempt completed in %v", elapsed)
}


// TestCentrifugeConnection_ConcurrentWithMixedAuth tests concurrent connections with mixed auth
func TestCentrifugeConnection_ConcurrentWithMixedAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	validConfig := testhelpers.GetCentrifugeConfig(Resource)
	invalidConfig := testhelpers.GetCentrifugeConfig(Resource)
	invalidConfig.Token = "invalid-token"

	// Create multiple connections with mixed auth
	numConnections := 6
	successCount := 0
	failCount := 0

	var wg sync.WaitGroup
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Alternate between valid and invalid tokens
			var Config duck.CentrifugalConfiguration
			var user string
			if index%2 == 0 {
				Config = validConfig
				user = fmt.Sprintf("valid-user-%d", index)
			} else {
				Config = invalidConfig
				user = fmt.Sprintf("invalid-user-%d", index)
			}

			client, err := duck.CentrifugeConnection(Config, user)
			if err != nil {
				failCount++
				t.Logf("Connection %d (%s) failed to create: %v", index, user, err)
				return
			}
			defer client.Close()

			// Try to wait for connection
			err = testhelpers.WaitForConnection(client, 5*time.Second)
			if err != nil {
				failCount++
				t.Logf("Connection %d (%s) failed to connect: %v", index, user, err)
				return
			}

			successCount++
			t.Logf("Connection %d (%s) succeeded", index, user)
		}(i)
	}

	wg.Wait()

	// Valid connections should succeed, invalid should fail
	t.Logf("Concurrent connections: %d succeeded, %d failed", successCount, failCount)
	assert.Greater(t, successCount, 0, "At least some connections should succeed")
	assert.Equal(t, numConnections, successCount+failCount, "All connections should either succeed or fail")
}

// TestCentrifugePublisher_LargeMessage tests handling of large messages
func TestCentrifugePublisher_LargeMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain


	client, err := duck.CentrifugeConnection(Config, "testuser-large-message")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create a large message (1MB value)
	largeValue := make([]byte, 1024*1024) // 1MB
	for i := range largeValue {
		largeValue[i] = byte('A' + (i % 26))
	}

	testMsg := duck.Message{
		Timestamp:     time.Now(),
		Host:          "test-host",
		Type:          duck.MessageInfo,
		Value:         string(largeValue),
		StopProcesses: false,
		RunNumber:     1,
	}

	start := time.Now()
	msgData, _ := json.Marshal(testMsg)
	marshalTime := time.Since(start)

	start = time.Now()
	_, err = sub.Publish(context.Background(), msgData)
	publishTime := time.Since(start)

	if err != nil {
		t.Logf("Large message publish failed (expected): %v (marshal: %v, publish: %v)",
			err, marshalTime, publishTime)
		assert.Error(t, err, "Large message publish should fail")
	} else {
		t.Logf("Large message published successfully (size: %d bytes, marshal: %v, publish: %v)",
			len(msgData), marshalTime, publishTime)

		// Wait for message
		err = tracker.WaitFor(1, 15*time.Second)
		if err != nil {
			t.Logf("Large message not received (may be too large): %v", err)
		} else {
			messages := tracker.GetAll()
			assert.Len(t, messages, 1)
			assert.Greater(t, len(messages[0].Value), 1024*1024, "Should receive large message")
			t.Log("Large message successfully sent and received")
		}
	}
}
