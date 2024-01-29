package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/centrifugal/centrifuge-go"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

// GetCentrifugeConfig returns configuration for test Centrifuge
func GetCentrifugeConfig(resource *dockertest.Resource) duck.CentrifugalConfiguration {
	host := getCentrifugeHost(resource)
	port := 8000
	if host == "127.0.0.1" {
		port = getResourcePort(resource, "8000/tcp")
	}
	return duck.CentrifugalConfiguration{
		Host:  host,
		Port:  port,
		Token: "testsecret",
	}
}

// getCentrifugeHost returns the appropriate host to connect to Centrifuge container
// When running inside Docker with host networking, use localhost with mapped port
func getCentrifugeHost(resource *dockertest.Resource) string {
	// Always use localhost - with host networking in the test container,
	// published ports are accessible on 127.0.0.1
	return "127.0.0.1"
}

// getResourcePort extracts the port from a dockertest Resource
func getResourcePort(resource *dockertest.Resource, portSpec string) int {
	portStr := resource.GetPort(portSpec)

	// dockertest returns just the host port as string (e.g., "32824")
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	// Validate that we got a valid port
	if port > 0 && port <= 65535 {
		return port
	}

	// Fallback to default
	return 8000
}

// ============================================
// NEW: Enhanced synchronization helpers
// ============================================

// WaitForConnection waits for Centrifuge client to connect with timeout
func WaitForConnection(client *centrifuge.Client, timeout time.Duration) error {
	errChan := make(chan error, 1)

	client.OnConnected(func(centrifuge.ConnectedEvent) {
		select {
		case errChan <- nil:
		default:
		}
	})

	client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		select {
		case errChan <- fmt.Errorf("client disconnected: %v", e):
		default:
		}
	})

	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("connection timeout after %v", timeout)
	}
}

// WaitForSubscriptionReady waits for subscription to be ready with timeout
func WaitForSubscriptionReady(sub *centrifuge.Subscription, timeout time.Duration) error {
	errChan := make(chan error, 1)

	sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		select {
		case errChan <- nil:
		default:
		}
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		select {
		case errChan <- fmt.Errorf("subscription error: %w", e.Error):
		default:
		}
	})

	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("subscription timeout after %v", timeout)
	}
}

// MessageTracker tracks received messages in a thread-safe manner
type MessageTracker struct {
	messages []*duck.Message
	mutex    sync.RWMutex
	cond     *sync.Cond
}

// NewMessageTracker creates a new message tracker
func NewMessageTracker() *MessageTracker {
	mt := &MessageTracker{
		messages: make([]*duck.Message, 0),
	}
	mt.cond = sync.NewCond(&mt.mutex)
	return mt
}

// Add adds a message to the tracker
func (mt *MessageTracker) Add(msg *duck.Message) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.messages = append(mt.messages, msg)
	mt.cond.Broadcast()
}

// GetAll returns all messages
func (mt *MessageTracker) GetAll() []*duck.Message {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()
	result := make([]*duck.Message, len(mt.messages))
	copy(result, mt.messages)
	return result
}

// WaitFor waits for at least count messages with timeout
func (mt *MessageTracker) WaitFor(count int, timeout time.Duration) error {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for len(mt.messages) < count && time.Now().Before(deadline) {
		mt.mutex.Unlock()
		select {
		case <-ticker.C:
			// Check again
		case <-time.After(time.Until(deadline)):
			mt.mutex.Lock()
			if len(mt.messages) >= count {
				return nil
			}
			return fmt.Errorf("timeout waiting for %d messages: got %d", count, len(mt.messages))
		}
		mt.mutex.Lock()
	}

	if len(mt.messages) >= count {
		return nil
	}
	return fmt.Errorf("timeout waiting for %d messages: got %d", count, len(mt.messages))
}

// GenerateTestUserNames generates unique test usernames
func GenerateTestUserNames(count int) []string {
	users := make([]string, count)
	for i := 0; i < count; i++ {
		users[i] = fmt.Sprintf("testuser-%d", i)
	}
	return users
}

func SetupCentrifugeForSystemTest(t *testing.T) (*dockertest.Pool, *dockertest.Resource, duck.CentrifugalConfiguration) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "Could not construct pool")

	err = pool.Client.Ping()
	require.NoError(t, err, "Could not connect to Docker")

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "centrifugo/centrifugo",
		Tag:        "v5",
		Env: []string{
			"CENTRIFUGO_TOKEN_HMAC_SECRET_KEY=testsecret",
			"CENTRIFUGO_ALLOW_SUBSCRIBE_FOR_CLIENT=true",
			"CENTRIFUGO_ALLOW_PUBLISH_FOR_CLIENT=true",
			"CENTRIFUGO_WEBSOCKET_CHECK_ORIGIN=false",
		},
		ExposedPorts: []string{"8000/tcp"},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		config.PortBindings = map[docker.Port][]docker.PortBinding{
			"8000/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}},
		}
	})
	require.NoError(t, err, "Could not start Centrifuge container")

	err = resource.Expire(120)
	require.NoError(t, err)

	t.Logf("Waiting for Centrifuge container to be ready...")
	pool.MaxWait = 30 * time.Second

	var centrifugeConfig duck.CentrifugalConfiguration

	err = pool.Retry(func() error {
		host := getCentrifugeHost(resource)
		port := "8000"

		if host != "127.0.0.1" {
			t.Logf("Connecting to Centrifuge via container IP: %s:%s", host, port)
		} else {
			mappedPort := resource.GetPort("8000/tcp")
			if mappedPort == "" {
				return fmt.Errorf("port not available")
			}
			port = strings.Split(mappedPort, "/")[0]
			t.Logf("Connecting to Centrifuge via localhost: %s:%s", host, port)
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 2*time.Second)
		if err != nil {
			t.Logf("Connection failed: %v", err)
			return err
		}
		conn.Close()

		t.Logf("Successfully connected to Centrifuge on %s:%s", host, port)

		portInt := 8000
		if host == "127.0.0.1" {
			mappedPort := resource.GetPort("8000/tcp")
			if mappedPort != "" {
				fmt.Sscanf(strings.Split(mappedPort, "/")[0], "%d", &portInt)
			}
		}

		centrifugeConfig = duck.CentrifugalConfiguration{
			Host:  host,
			Port:  portInt,
			Token: "testsecret",
		}

		time.Sleep(2 * time.Second)
		return nil
	})
	require.NoError(t, err, "Centrifuge did not become ready in time")

	return pool, resource, centrifugeConfig
}

type CentrifugeSystemTracker struct {
	t        *testing.T
	messages []duck.Message
	mu       sync.Mutex
	sub      *centrifuge.Subscription
	client   *centrifuge.Client
}

func NewCentrifugeSystemTracker(t *testing.T, cfg duck.CentrifugalConfiguration) *CentrifugeSystemTracker {
	address := fmt.Sprintf("ws://%s:%d/connection/websocket", cfg.Host, cfg.Port)

	token := duck.ConnToken("test-tracker", 0, cfg.Token)

	clientConfig := centrifuge.Config{
		Token: token,
	}

	client := centrifuge.NewJsonClient(address, clientConfig)

	tracker := &CentrifugeSystemTracker{
		t:        t,
		messages: make([]duck.Message, 0),
		client:   client,
	}

	client.OnError(func(e centrifuge.ErrorEvent) {
		t.Logf("Centrifuge client error: %v", e.Error)
	})

	sub, err := client.NewSubscription(duck.CENTRIFUGE_TOPIC, centrifuge.SubscriptionConfig{
		Recoverable: true,
		JoinLeave:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}
	tracker.sub = sub

	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		var msg duck.Message
		if err := json.Unmarshal(e.Data, &msg); err != nil {
			t.Logf("Failed to unmarshal message: %v", err)
			return
		}
		tracker.mu.Lock()
		tracker.messages = append(tracker.messages, msg)
		tracker.mu.Unlock()
		t.Logf("Received Centrifuge message: type=%s, host=%s, stop_processes=%v, value=%.100s",
			msg.Type, msg.Host, msg.StopProcesses, msg.Value)
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		t.Logf("Subscription error: %v", e.Error)
	})

	err = sub.Subscribe()
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connected := false
	client.OnConnected(func(e centrifuge.ConnectedEvent) {
		connected = true
		t.Logf("CentrifugeSystemTracker connected")
	})

	for !connected {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for Centrifuge connection")
		case <-time.After(100 * time.Millisecond):
		}
	}

	t.Logf("CentrifugeSystemTracker connected and subscribed to topic '%s'", duck.CENTRIFUGE_TOPIC)

	return tracker
}

func (t *CentrifugeSystemTracker) WaitForErrorMessage(timeout time.Duration) (*duck.Message, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		t.mu.Lock()
		for i := len(t.messages) - 1; i >= 0; i-- {
			if t.messages[i].Type == duck.MessageError {
				msg := t.messages[i]
				t.mu.Unlock()
				return &msg, nil
			}
		}
		t.mu.Unlock()

		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("no error message received within timeout")
}

func (t *CentrifugeSystemTracker) WaitForStopProcessesMessage(timeout time.Duration) (*duck.Message, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		t.mu.Lock()
		for i := len(t.messages) - 1; i >= 0; i-- {
			if t.messages[i].Type == duck.MessageError && t.messages[i].StopProcesses {
				msg := t.messages[i]
				t.mu.Unlock()
				return &msg, nil
			}
		}
		t.mu.Unlock()

		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("no stop_processes error message received within timeout")
}

func (t *CentrifugeSystemTracker) GetAll() []duck.Message {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]duck.Message, len(t.messages))
	copy(result, t.messages)
	return result
}

func (t *CentrifugeSystemTracker) Close() {
	if t.sub != nil {
		t.sub.Unsubscribe()
	}
	if t.client != nil {
		t.client.Disconnect()
	}
}
