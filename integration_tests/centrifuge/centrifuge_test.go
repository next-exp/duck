//go:build integration
// +build integration

package centrifuge_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

var (
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
	Config   duck.CentrifugalConfiguration
)

func TestMain(m *testing.M) {
	// Setup: Create one shared Centrifuge container for all tests
	Pool, Resource, Config = testhelpers.SetupCentrifugeContainerForTestMain()

	// Run all tests
	code := m.Run()

	// Teardown: Cleanup the shared container
	if Resource != nil {
		if err := Pool.Purge(Resource); err != nil {
			fmt.Printf("Warning: Failed to purge resource: %v\n", err)
		}
	}

	os.Exit(code)
}
