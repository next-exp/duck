//go:build integration
// +build integration

package ldc_test

import (
	"os"
	"testing"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
)

// TestMain sets up a shared MySQL container for all LDC tests
// This significantly reduces test time by avoiding container startup overhead for each test
func TestMain(m *testing.M) {
	// Setup shared MySQL container for all tests in this package
	testhelpers.SetupSharedMySQL()

	// Run tests
	code := m.Run()

	// Teardown shared MySQL container
	testhelpers.TeardownSharedMySQL()

	os.Exit(code)
}
