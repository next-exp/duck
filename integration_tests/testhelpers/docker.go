package testhelpers

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// getContainerHost returns the appropriate host to connect to a container
// When running inside Docker (with DOCKER_HOST set or in a container), use container IP
// Otherwise use localhost with mapped port
func getContainerHost(resource *dockertest.Resource) string {
	// With host networking in the test container, always use localhost
	// The published ports will be accessible on 127.0.0.1
	return "127.0.0.1"
}

// getMySQLOptions returns standard Docker run options for MySQL containers
func getMySQLOptions() *dockertest.RunOptions {
	return &dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "8.0",
		Env: []string{
			"MYSQL_ROOT_PASSWORD=testpass",
			"MYSQL_DATABASE=duck_test",
			"MYSQL_USER=testuser",
			"MYSQL_PASSWORD=testpass",
		},
		Cmd: []string{"--default-authentication-plugin=mysql_native_password"},
	}
}

// SetupCentrifugeContainer starts Centrifuge server
func SetupCentrifugeContainer(t testing.TB) (*dockertest.Pool, *dockertest.Resource) {
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

	err = resource.Expire(30)
	require.NoError(t, err)

	// Wait for Centrifuge to be ready
	t.Logf("Waiting for Centrifuge container to be ready...")
	pool.MaxWait = 30 * time.Second
	err = pool.Retry(func() error {
		// Debug: log the resource info
		t.Logf("Container ID: %s", resource.Container.ID)
		t.Logf("Container name: %s", resource.Container.Name)

		host := getContainerHost(resource)
		port := "8000"
		// If using container IP, use internal port. Otherwise use mapped port
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

		// Just check if we can establish a connection
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 2*time.Second)
		if err != nil {
			t.Logf("Connection failed: %v", err)
			return err
		}
		conn.Close()

		t.Logf("Successfully connected to Centrifuge on %s:%s", host, port)

		// Give Centrifuge server a moment to fully initialize
		time.Sleep(2 * time.Second)
		return nil
	})
	require.NoError(t, err, "Centrifuge did not become ready in time")

	return pool, resource
}

// CleanupDocker removes containers
func CleanupDocker(pool *dockertest.Pool, resources ...*dockertest.Resource) {
	for _, resource := range resources {
		if resource != nil {
			if err := pool.Purge(resource); err != nil {
				// Log but don't fail tests on cleanup error
				fmt.Printf("Warning: Failed to purge resource %s: %v\n", resource.Container.Name, err)
			}
		}
	}
}

// SetupCentrifugeContainerForTestMain creates a Centrifuge container for TestMain
// This function doesn't require testing.TB and uses fmt.Printf for logging
func SetupCentrifugeContainerForTestMain() (*dockertest.Pool, *dockertest.Resource, duck.CentrifugalConfiguration) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(fmt.Sprintf("Could not construct pool: %v", err))
	}

	err = pool.Client.Ping()
	if err != nil {
		panic(fmt.Sprintf("Could not connect to Docker: %v", err))
	}

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
	if err != nil {
		panic(fmt.Sprintf("Could not start Centrifuge container: %v", err))
	}

	err = resource.Expire(120)
	if err != nil {
		panic(fmt.Sprintf("Could not set resource expiration: %v", err))
	}

	// Wait for Centrifuge to be ready
	fmt.Println("Waiting for Centrifuge container to be ready...")
	pool.MaxWait = 30 * time.Second
	err = pool.Retry(func() error {
		host := getContainerHost(resource)
		port := "8000"
		if host == "127.0.0.1" {
			mappedPort := resource.GetPort("8000/tcp")
			if mappedPort == "" {
				return fmt.Errorf("port not available")
			}
			port = mappedPort
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 2*time.Second)
		if err != nil {
			return err
		}
		conn.Close()

		time.Sleep(2 * time.Second)
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("Centrifuge did not become ready: %v", err))
	}

	config := GetCentrifugeConfig(resource)
	return pool, resource, config
}
