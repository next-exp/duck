//go:build integration
// +build integration

package e2e_test

import (
	"os"
	"testing"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
)

func TestMain(m *testing.M) {
	testhelpers.SetupSharedMySQL()
	code := m.Run()
	testhelpers.TeardownSharedMySQL()
	os.Exit(code)
}
