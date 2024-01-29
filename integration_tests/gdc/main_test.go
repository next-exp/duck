//go:build integration
// +build integration

package gdc_test

import (
	"os"
	"testing"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
)

// TestMain sets up a shared MySQL container for all GDC integration tests
// This significantly reduces test time by avoiding container startup overhead for each test
// Note: GDC itself doesn't need MySQL, but tests that use StartTestLDC do
func TestMain(m *testing.M) {
	// Setup shared MySQL container for all tests in this package
	testhelpers.SetupSharedMySQL()

	// Run tests
	code := m.Run()

	// Teardown shared MySQL container
	testhelpers.TeardownSharedMySQL()

	os.Exit(code)
}
