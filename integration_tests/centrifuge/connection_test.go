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

	"github.com/centrifugal/centrifuge-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// TestCentrifugeConnection_ConnectsSuccessfully tests that CentrifugeConnection creates a valid client
func TestCentrifugeConnection_ConnectsSuccessfully(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Test connection with user
	client, err := duck.CentrifugeConnection(Config, "testuser-connect-success")
	require.NoError(t, err, "Should connect successfully")
	require.NotNil(t, client, "Client should not be nil")
	defer client.Close()

	// Wait for connection to establish using proper synchronization
	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err, "Should connect within 5 seconds")

	// Verify client is connected by checking its state
	// The centrifuge client doesn't expose a simple IsConnected() method,
	// but we can verify it was created without error and hasn't disconnected
	assert.NotNil(t, client, "Client should remain valid after connection")
}

// TestCentrifugeConnection_WithInvalidHost tests connection failure with invalid host
func TestCentrifugeConnection_WithInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := duck.CentrifugalConfiguration{
		Host:  "invalid-host-that-does-not-exist.local",
		Port:  9999,
		Token: "testsecret",
	}

	// This should either fail immediately or timeout
	client, err := duck.CentrifugeConnection(config, "testuser")

	// The connection might succeed initially but fail during Connect()
	// or it might fail during client creation
	if err == nil {
		defer client.Close()
		// If we got a client, try to connect - this should fail
		err = client.Connect()
		require.Error(t, err, "Connect should fail with invalid host")
	} else {
		// Expected: connection failed
		require.Error(t, err, "Should fail to connect to invalid host")
	}
}

// TestSubscribeCentrifuge_WithNilCallback tests subscription without callback
func TestSubscribeCentrifuge_WithNilCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	client, err := duck.CentrifugeConnection(Config, "testuser-connect-success")
	require.NoError(t, err)
	defer client.Close()

	// Wait for connection
	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	// Subscribe with nil callback - should still work
	sub, err := duck.SubscribeCentrifuge(client, nil)
	require.NoError(t, err, "Should subscribe even with nil callback")
	require.NotNil(t, sub)
	defer sub.Unsubscribe()

	// Wait for subscription to be ready
	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Subscription should exist but won't process messages
	assert.NotNil(t, sub, "Subscription should be created")
}

// TestCreateNewSubscriptionWithFnReadout_CreatesValidSubscription tests CreateNewSubscriptionWithFnReadout
func TestCreateNewSubscriptionWithFnReadout_CreatesValidSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Track received messages using message tracker
	tracker := testhelpers.NewMessageTracker()

	sub, err := duck.CreateNewSubscriptionWithFnReadout(Config, "test-service", func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err, "Should create subscription with callback successfully")
	require.NotNil(t, sub)
	defer sub.Unsubscribe()

	// Wait for subscription to be established
	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Publish multiple test messages
	expectedCount := 3
	for i := 0; i < expectedCount; i++ {
		testMsg := duck.Message{
			Timestamp:     time.Now(),
			Host:          "test-service",
			Type:          duck.MessageState,
			Value:         "test-state",
			StopProcesses: false,
			RunNumber:     i + 1,
		}

		msgData, err := json.Marshal(testMsg)
		require.NoError(t, err)

		_, err = sub.Publish(context.Background(), msgData)
		require.NoError(t, err)
	}

	// Wait for all messages to be received using tracker
	err = tracker.WaitFor(expectedCount, 10*time.Second)
	require.NoError(t, err, fmt.Sprintf("Should receive %d messages within 10 seconds", expectedCount))

	// Verify we received all messages
	messages := tracker.GetAll()
	assert.Len(t, messages, expectedCount, "Should receive all messages")
}

// TestPublishMessage_WithRealCentrifuge tests PublishMessage with real Centrifuge
func TestPublishMessage_WithRealCentrifuge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Create publisher
	pubSub, err := duck.CreateNewSubscription(Config, "publisher")
	require.NoError(t, err)
	defer pubSub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(pubSub, 5*time.Second)
	require.NoError(t, err)

	publisher := &duck.CentrifugePublisher{Sub: pubSub}

	// Create subscriber
	tracker := testhelpers.NewMessageTracker()

	subClient, err := duck.CentrifugeConnection(Config, "subscriber")
	require.NoError(t, err)
	defer subClient.Close()

	err = testhelpers.WaitForConnection(subClient, 5*time.Second)
	require.NoError(t, err)

	_, err = duck.SubscribeCentrifuge(subClient, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)

	// Publish using PublishMessage function
	testMsg := duck.Message{
		Timestamp:     time.Now(),
		Host:          "test-publisher",
		Type:          duck.MessageError,
		Value:         "test error message",
		StopProcesses: true,
		RunNumber:     42,
	}

	err = duck.PublishMessage(publisher, testMsg)
	require.NoError(t, err, "PublishMessage should succeed")

	// Wait for subscriber to receive
	err = tracker.WaitFor(1, 5*time.Second)
	require.NoError(t, err, "Subscriber should receive message within 5 seconds")

	messages := tracker.GetAll()
	require.Len(t, messages, 1, "Subscriber should have received message")

	receivedMsg := messages[0]
	assert.Equal(t, duck.MessageError, receivedMsg.Type)
	assert.Equal(t, "test error message", receivedMsg.Value)
	assert.True(t, receivedMsg.StopProcesses, "StopProcesses flag should be preserved")
	assert.Equal(t, 42, receivedMsg.RunNumber)
}

// TestSubscribeCentrifuge_ReceivesMultipleMessageTypes tests receiving various message types
func TestSubscribeCentrifuge_ReceivesMultipleMessageTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	client, err := duck.CentrifugeConnection(Config, "testuser-connect-success")
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

	// Publish different message types
	messageTypes := []struct {
		msgType duck.MessageType
		value   string
	}{
		{duck.MessageInfo, "info message"},
		{duck.MessageError, "error message"},
		{duck.MessageMetric, "metric message"},
		{duck.MessageState, "state message"},
		{duck.MessageDebug, "debug message"},
	}

	for _, mt := range messageTypes {
		testMsg := duck.Message{
			Timestamp:     time.Now(),
			Host:          "test-host",
			Type:          mt.msgType,
			Value:         mt.value,
			StopProcesses: false,
			RunNumber:     1,
		}

		msgData, err := json.Marshal(testMsg)
		require.NoError(t, err)

		_, err = sub.Publish(context.Background(), msgData)
		require.NoError(t, err)
	}

	// Wait for all messages
	err = tracker.WaitFor(len(messageTypes), 10*time.Second)
	require.NoError(t, err, fmt.Sprintf("Should receive %d messages", len(messageTypes)))

	// Verify we got all message types
	messages := tracker.GetAll()
	assert.Len(t, messages, len(messageTypes), "Should receive all message types")

	// Check that each type was received
	receivedTypes := make(map[duck.MessageType]bool)
	for _, msg := range messages {
		receivedTypes[msg.Type] = true
	}

	for _, mt := range messageTypes {
		assert.True(t, receivedTypes[mt.msgType], "Should have received message type %s", mt.msgType)
	}
}

// TestCentrifugeConnection_MultipleConnections tests multiple simultaneous connections
func TestCentrifugeConnection_MultipleConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Create multiple connections
	numConnections := 5
	clients := make([]*centrifuge.Client, numConnections)
	connectionErrors := make(chan error, numConnections)

	var wg sync.WaitGroup
	users := testhelpers.GenerateTestUserNames(numConnections)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			user := users[index]
			client, err := duck.CentrifugeConnection(Config, user)
			if err != nil {
				connectionErrors <- err
				return
			}
			clients[index] = client

			// Wait for connection to be established
			err = testhelpers.WaitForConnection(client, 5*time.Second)
			connectionErrors <- err
		}(i)
	}

	wg.Wait()
	close(connectionErrors)

	// Check all connections succeeded
	for err := range connectionErrors {
		require.NoError(t, err, "All connections should succeed")
	}

	// Verify all clients were created
	for i, client := range clients {
		assert.NotNil(t, client, "Client %d should not be nil", i)
		if client != nil {
			defer client.Close()
		}
	}
}

// TestCreateNewSubscription_MultipleSubscriptions tests multiple subscriptions
func TestCreateNewSubscription_MultipleSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container from TestMain

	// Create multiple subscriptions
	numSubscriptions := 3
	subscriptions := make([]*centrifuge.Subscription, numSubscriptions)
	users := testhelpers.GenerateTestUserNames(numSubscriptions)

	for i := 0; i < numSubscriptions; i++ {
		user := users[i]
		sub, err := duck.CreateNewSubscription(Config, user)
		require.NoError(t, err, "Subscription %d should succeed", i)
		require.NotNil(t, sub, "Subscription %d should not be nil", i)
		subscriptions[i] = sub
		defer sub.Unsubscribe()

		// Wait for subscription to be ready
		err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
		require.NoError(t, err, "Subscription %d should be ready", i)
	}

	// Verify all subscriptions are valid
	for i, sub := range subscriptions {
		assert.NotNil(t, sub, "Subscription %d should remain valid", i)
	}
}
