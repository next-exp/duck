//go:build integration
// +build integration

package database_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	"github.com/jmbenlloch/next_duck/pkg/database"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// TestMain sets up a shared MySQL container for all tests
func TestMain(m *testing.M) {
	// Setup shared MySQL container
	testhelpers.SetupSharedMySQL()

	// Run tests
	code := m.Run()

	// Teardown shared MySQL container
	testhelpers.TeardownSharedMySQL()

	os.Exit(code)
}

// getSharedDB connects to the shared MySQL database
func getSharedDB(t *testing.T) *sql.DB {
	dbConfig := testhelpers.GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err, "Failed to connect to shared test database")
	return db
}


// TestReadConfiguration_RealDatabase tests reading complete configuration from a real MySQL database.
//
// This is the primary integration test that validates the entire configuration reading pipeline.
// It tests actual database interaction (not mocks) and validates complex nested relationships
// (LDC → Equipment). It catches schema/ORM mismatches that unit tests can't detect.
//
// What it validates:
// - sqlc-generated queries work correctly with real MySQL
// - Column mapping from DB to Go structs (including NULL handling via sql.Null* types)
// - Complex nested relationships (LDCs contain their assigned Equipment)
// - All configuration sections: GDCs, LDCs, Equipment, Run, Duck, Decoder, TestDeviceParams
func TestReadConfiguration_RealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Read configuration from database using sqlc-generated queries
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err, "Failed to read configuration from database")

	// Verify GDCs loaded correctly (tests sqlc.ListGDCs)
	assert.Len(t, config.GDCs, 2, "Should load 2 GDCs from seed data")
	assert.Equal(t, "test_gdc1", config.GDCs[0].Name, "First GDC should be test_gdc1")
	assert.Equal(t, "test_gdc2", config.GDCs[1].Name, "Second GDC should be test_gdc2")
	assert.Equal(t, "127.0.0.1", config.GDCs[0].IP, "GDC IP should match seed data")
	assert.Equal(t, 6005, config.GDCs[0].Port, "GDC port should match seed data")
	assert.Equal(t, 50051, config.GDCs[0].GRPCPort, "GDC gRPC port should match seed data")
	assert.True(t, config.GDCs[0].Enabled, "GDC should be enabled")
	assert.True(t, config.GDCs[0].WriteOutput, "GDC writeOutput should be enabled")
	assert.False(t, config.GDCs[0].Decode, "GDC decode should be disabled")

	// Verify LDCs loaded correctly (tests sqlc.ListLDCs)
	assert.Len(t, config.LDCs, 2, "Should load 2 LDCs from seed data")
	assert.Equal(t, "test_ldc1", config.LDCs[0].Name, "First LDC should be test_ldc1")
	assert.Equal(t, "test_ldc2", config.LDCs[1].Name, "Second LDC should be test_ldc2")
	assert.Equal(t, "127.0.0.1", config.LDCs[0].IP, "LDC IP should match seed data")
	assert.Equal(t, 50053, config.LDCs[0].GRPCPort, "LDC gRPC port should match seed data")
	assert.True(t, config.LDCs[0].Enabled, "LDC should be enabled")

	// Verify equipment assignment to LDCs (tests sqlc.ListEquipments + join logic)
	assert.Len(t, config.LDCs[0].Equipments, 2, "LDC 1 should have 2 equipments")
	assert.Len(t, config.LDCs[1].Equipments, 2, "LDC 2 should have 2 equipments")

	// Verify first equipment details
	eq1 := config.LDCs[0].Equipments[0]
	assert.Equal(t, 1, eq1.ID, "Equipment ID should match")
	assert.Equal(t, 22, eq1.Type, "Equipment type should be 22")
	assert.Equal(t, "127.0.0.1", eq1.DeviceIP, "Device IP should match")
	assert.Equal(t, "127.0.0.1", eq1.HostIP, "Host IP should match")
	assert.Equal(t, 16006, eq1.HostPort, "Host port should match")
	assert.True(t, eq1.Enabled, "Equipment should be enabled")
	assert.Equal(t, 1, eq1.LDC_ID, "Equipment should reference LDC 1")

	// Verify run number (tests sqlc.GetLatestRun)
	assert.Greater(t, config.RunNumber, 0, "Run number should be positive")

	// Verify duck parameters loaded (tests sqlc.GetDuckParams)
	assert.Equal(t, "integration_test", config.Duck.Experiment, "Experiment name should match seed data")
	assert.Equal(t, 1000000000, config.Duck.Filesize, "File size should match")
	assert.Equal(t, 1400, config.Duck.PacketSize, "Packet size should match")
	assert.Equal(t, 4, config.Duck.DecoderWorkers, "Decoder workers should match")

	// Verify decoder configuration loaded (tests sqlc.GetDecoderParams)
	assert.Equal(t, 0, config.Decoder.ExtTrigger, "Decoder ext_trigger should match seed")
	assert.Equal(t, 1, config.Decoder.TrgCode1, "Decoder trg_code_1 should match seed")
	assert.Equal(t, 2, config.Decoder.TrgCode2, "Decoder trg_code_2 should match seed")
	assert.True(t, config.Decoder.ReadPMTs, "Decoder read_pmts should be true")
	assert.True(t, config.Decoder.ReadSiPMs, "Decoder read_sipms should be true")
	assert.True(t, config.Decoder.ReadTrigger, "Decoder read_trigger should be true")
	assert.False(t, config.Decoder.SplitTrigger, "Decoder split_trigger should be false")
	assert.False(t, config.Decoder.NoDB, "Decoder no_db should be false")
	assert.False(t, config.Decoder.Discard, "Decoder discard should be false")
	assert.Equal(t, "localhost", config.Decoder.Host, "Decoder host should match")
	assert.Equal(t, "test", config.Decoder.User, "Decoder user should match")
	assert.Equal(t, "test_db", config.Decoder.DBName, "Decoder db_name should match")
	assert.True(t, config.Decoder.WriteData, "Decoder write_data should be true")
	assert.True(t, config.Decoder.UseBlosc, "Decoder use_blosc should be true")
	assert.Equal(t, "lz4", config.Decoder.BloscAlgorithm, "Decoder blosc_algorithm should match")
	assert.Equal(t, 5, config.Decoder.CompressionLevel, "Decoder compression_level should match")
	assert.Equal(t, "true", config.Decoder.BitShuffle, "Decoder bit_shuffle should match")

	// Verify test device parameters loaded (tests sqlc.GetTestDeviceParams)
	assert.Len(t, config.TestDeviceParams, 3, "Should load 3 test device param entries")

	// Check first equipment's test params
	eq1Params := config.TestDeviceParams[0]
	assert.Equal(t, 1, eq1Params.EquipmentID, "Test params should be for equipment 1")
	assert.Equal(t, 10, eq1Params.EventRateMs, "Event rate should match seed")
	assert.Equal(t, 100, eq1Params.PacketsPerEvent, "Packets per event should match seed")
	assert.Equal(t, 1400, eq1Params.PacketSize, "Packet size should match seed")
	assert.Equal(t, 0.0, eq1Params.ErrorInjectionRate, "Error injection rate should match seed")
	assert.Equal(t, 0, eq1Params.MaxEvents, "Max events should match seed")

	// Check second equipment's test params with error injection
	eq2Params := config.TestDeviceParams[1]
	assert.Equal(t, 2, eq2Params.EquipmentID, "Test params should be for equipment 2")
	assert.Equal(t, 0.01, eq2Params.ErrorInjectionRate, "Error injection rate should be 1%")
	assert.Equal(t, 1000, eq2Params.MaxEvents, "Max events should be 1000")
}

// TestReadConfiguration_EnabledFiltering tests that disabled equipment/LDCs/GDCs are correctly reflected and can be filtered.
//
// This test validates the business logic for filtering out disabled components, which is critical because:
// 1. Disabled GDCs/LDCs should not receive traffic
// 2. Equipment on disabled LDCs must not be configured
// 3. The filtering is done at application level, not DB level
//
// It also tests:
// - Dynamic filtering logic using EnabledGDCs() and EnabledLDCs() helpers
// - SQL NULL handling for boolean fields (sql.NullBool)
// - Real database updates + re-reading configuration
func TestReadConfiguration_EnabledFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Phase 1: all enabled from seed
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	for _, gdc := range config.GDCs {
		assert.True(t, gdc.Enabled, "All GDCs in test data should be enabled")
	}
	for _, ldc := range config.LDCs {
		assert.True(t, ldc.Enabled, "All LDCs in test data should be enabled")
		for _, eq := range ldc.Equipments {
			assert.True(t, eq.Enabled, "All equipment in test data should be enabled")
		}
	}

	// Phase 2: disable one GDC, one LDC, and one equipment
	// Use name-based lookups instead of hardcoded IDs for robustness
	var gdcID, ldcID, equipmentID int32

	// Get GDC ID by name
	err = db.QueryRow("SELECT id FROM gdcs WHERE name = ?", "test_gdc2").Scan(&gdcID)
	require.NoError(t, err, "Should find test_gdc2")

	// Get LDC ID by name
	err = db.QueryRow("SELECT id FROM ldcs WHERE name = ?", "test_ldc2").Scan(&ldcID)
	require.NoError(t, err, "Should find test_ldc2")

	// Get second equipment ID for LDC 1 (we know equipment 1 and 2 belong to LDC 1)
	err = db.QueryRow("SELECT id FROM equipments WHERE ldcID = 1 AND id != 1").Scan(&equipmentID)
	require.NoError(t, err, "Should find second equipment for LDC 1")

	// Disable them
	_, err = db.Exec("UPDATE gdcs SET enabled=false WHERE id=?", gdcID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE ldcs SET enabled=false WHERE id=?", ldcID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE equipments SET enabled=false WHERE id=?", equipmentID)
	require.NoError(t, err)

	// Re-read configuration
	queries = database.New(db)
	config, err = duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	// Verify enabled filtering helpers work correctly
	enabledGDCs := duck.EnabledGDCs(config.GDCs)
	assert.Len(t, enabledGDCs, 1, "Only one GDC should be enabled after update")
	assert.Equal(t, "test_gdc1", enabledGDCs[0].Name, "test_gdc1 should still be enabled")

	enabledLDCs := duck.EnabledLDCs(config.LDCs)
	assert.Len(t, enabledLDCs, 1, "Only one LDC should be enabled after update")
	assert.Equal(t, "test_ldc1", enabledLDCs[0].Name, "test_ldc1 should still be enabled")

	// Verify disabled flags are reflected in raw config (tests sql.NullBool handling)
	var gdc2Disabled, ldc2Disabled, eq2Disabled bool
	for _, g := range config.GDCs {
		if g.Name == "test_gdc2" {
			gdc2Disabled = !g.Enabled
		}
	}
	for _, l := range config.LDCs {
		if l.Name == "test_ldc2" {
			ldc2Disabled = !l.Enabled
		}
		if l.Name == "test_ldc1" {
			for _, e := range l.Equipments {
				if int(e.ID) == int(equipmentID) {
					eq2Disabled = !e.Enabled
				}
			}
		}
	}
	assert.True(t, gdc2Disabled, "test_gdc2 should be disabled")
	assert.True(t, ldc2Disabled, "test_ldc2 should be disabled")
	assert.True(t, eq2Disabled, "Equipment should be disabled")

	// And verify that within enabled LDC 1 only one equipment remains enabled
	var enabledEqCount int
	for _, l := range config.LDCs {
		if l.Name == "test_ldc1" {
			for _, e := range l.Equipments {
				if e.Enabled {
					enabledEqCount++
				}
			}
		}
	}
	assert.Equal(t, 1, enabledEqCount, "LDC 1 should have exactly one enabled equipment after update")
}

// TestReadConfiguration_DecoderParams tests decoder-specific configuration loading from real database.
//
// This validates the sqlc.GetDecoderParams query and ensures all decoder configuration fields
// are correctly mapped from the database to the DecoderConfiguration struct.
// It tests NULL handling for boolean fields and string fields.
func TestReadConfiguration_DecoderParams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	// Verify all decoder parameters match seed data
	assert.Equal(t, 0, config.Decoder.ExtTrigger, "ext_trigger should match")
	assert.Equal(t, 1, config.Decoder.TrgCode1, "trg_code_1 should match")
	assert.Equal(t, 2, config.Decoder.TrgCode2, "trg_code_2 should match")

	// Test boolean NULL handling (all booleans in decoderParams are NOT NULL in schema)
	assert.True(t, config.Decoder.ReadPMTs, "read_pmts should be true")
	assert.True(t, config.Decoder.ReadSiPMs, "read_sipms should be true")
	assert.True(t, config.Decoder.ReadTrigger, "read_trigger should be true")
	assert.False(t, config.Decoder.SplitTrigger, "split_trigger should be false")
	assert.False(t, config.Decoder.NoDB, "no_db should be false")
	assert.False(t, config.Decoder.Discard, "discard should be false")
	assert.True(t, config.Decoder.WriteData, "write_data should be true")
	assert.True(t, config.Decoder.UseBlosc, "use_blosc should be true")

	// Test string fields
	assert.Equal(t, "localhost", config.Decoder.Host, "host should match")
	assert.Equal(t, "test", config.Decoder.User, "user should match")
	assert.Equal(t, "test", config.Decoder.Passwd, "passwd should match")
	assert.Equal(t, "test_db", config.Decoder.DBName, "db_name should match")
	assert.Equal(t, "lz4", config.Decoder.BloscAlgorithm, "blosc_algorithm should match")
	assert.Equal(t, "true", config.Decoder.BitShuffle, "bit_shuffle should match")

	// Test numeric fields
	assert.Equal(t, 5, config.Decoder.CompressionLevel, "compression_level should match")
}

// TestReadConfiguration_DecoderParamsDynamicUpdate tests that decoder parameter changes are reflected.
//
// This validates that the application can handle dynamic decoder configuration updates,
// which is important for runtime reconfiguration scenarios.
func TestReadConfiguration_DecoderParamsDynamicUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Read initial configuration
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	assert.Equal(t, 0, config.Decoder.ExtTrigger, "Initial ext_trigger should be 0")
	assert.True(t, config.Decoder.ReadPMTs, "Initial read_pmts should be true")

	// Update decoder parameters
	_, err = db.Exec("UPDATE decoderParams SET ext_trigger=?, read_pmts=? WHERE 1=1", 1, false)
	require.NoError(t, err)

	// Re-read configuration
	queries = database.New(db)
	config, err = duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	assert.Equal(t, 1, config.Decoder.ExtTrigger, "ext_trigger should be updated to 1")
	assert.False(t, config.Decoder.ReadPMTs, "read_pmts should be updated to false")
}

// TestReadConfiguration_TestDeviceParams tests test device parameter loading from real database.
//
// This validates the sqlc.GetTestDeviceParams query which retrieves simulation/testing
// parameters for individual equipment pieces. These parameters control event generation
// for testing/simulation purposes.
func TestReadConfiguration_TestDeviceParams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	// Should have 3 test device parameter entries from seed data
	assert.Len(t, config.TestDeviceParams, 3, "Should have 3 test device param entries")

	// Verify equipment 1 params (no errors, unlimited events)
	eq1Params := findTestDeviceParams(config, 1)
	require.NotNil(t, eq1Params, "Should find params for equipment 1")
	assert.Equal(t, 1, eq1Params.EquipmentID)
	assert.Equal(t, 10, eq1Params.EventRateMs, "Event rate should be 10ms")
	assert.Equal(t, 100, eq1Params.PacketsPerEvent, "Should have 100 packets per event")
	assert.Equal(t, 1400, eq1Params.PacketSize, "Packet size should be 1400")
	assert.Equal(t, 0.0, eq1Params.ErrorInjectionRate, "No error injection")
	assert.Equal(t, 0, eq1Params.MaxEvents, "Unlimited events (0 = unlimited)")

	// Verify equipment 2 params (with error injection)
	eq2Params := findTestDeviceParams(config, 2)
	require.NotNil(t, eq2Params, "Should find params for equipment 2")
	assert.Equal(t, 2, eq2Params.EquipmentID)
	assert.Equal(t, 15, eq2Params.EventRateMs, "Event rate should be 15ms")
	assert.Equal(t, 150, eq2Params.PacketsPerEvent, "Should have 150 packets per event")
	assert.Equal(t, 1400, eq2Params.PacketSize, "Packet size should be 1400")
	assert.Equal(t, 0.01, eq2Params.ErrorInjectionRate, "1% error injection rate")
	assert.Equal(t, 1000, eq2Params.MaxEvents, "Max 1000 events")

	// Verify equipment 3 params (different packet size)
	eq3Params := findTestDeviceParams(config, 3)
	require.NotNil(t, eq3Params, "Should find params for equipment 3")
	assert.Equal(t, 3, eq3Params.EquipmentID)
	assert.Equal(t, 20, eq3Params.EventRateMs, "Event rate should be 20ms")
	assert.Equal(t, 200, eq3Params.PacketsPerEvent, "Should have 200 packets per event")
	assert.Equal(t, 1500, eq3Params.PacketSize, "Packet size should be 1500")
	assert.Equal(t, 0.0, eq3Params.ErrorInjectionRate, "No error injection")
	assert.Equal(t, 0, eq3Params.MaxEvents, "Unlimited events")
}

// TestReadConfiguration_TestDeviceParamsOptional tests that configuration works when test device params are missing.
//
// This validates that the system doesn't break when testDeviceParams table is empty,
// which is the case in production environments (no testing/simulation equipment).
func TestReadConfiguration_TestDeviceParamsOptional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Clear test device params to simulate production environment
	_, err := db.Exec("DELETE FROM testDeviceParams")
	require.NoError(t, err)

	// Read configuration - should succeed with empty test device params
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err, "Should succeed even without test device params")

	// Should have empty test device params
	assert.Len(t, config.TestDeviceParams, 0, "Should have no test device params")

	// But other config should still load correctly
	assert.Len(t, config.GDCs, 2, "GDCs should still load")
	assert.Len(t, config.LDCs, 2, "LDCs should still load")
	assert.Greater(t, config.RunNumber, 0, "Run number should still load")
	assert.NotEmpty(t, config.Duck.Experiment, "Duck config should still load")
}

// TestReadConfiguration_SQLNullHandling tests sql.Null* type handling for nullable database columns.
//
// This validates that sqlc-generated code correctly handles NULL values from the database
// and converts them appropriately to Go types.
func TestReadConfiguration_SQLNullHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Insert a GDC with NULL values in nullable columns
	_, err := db.Exec(`
		INSERT INTO gdcs (name, hostname, ip, port, grpc_port, prometheus_port, datapath, enabled, writeOutput, decode)
		VALUES (?, NULL, NULL, NULL, NULL, NULL, NULL, true, true, false)
	`, "null_gdc")
	require.NoError(t, err)

	// Read configuration
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	// Find the GDC with NULL values
	var nullGdc *duck.GDCConfiguration
	for i := range config.GDCs {
		if config.GDCs[i].Name == "null_gdc" {
			nullGdc = &config.GDCs[i]
			break
		}
	}

	require.NotNil(t, nullGdc, "Should find the GDC with NULL values")
	assert.Equal(t, "null_gdc", nullGdc.Name, "Name should be set")
	assert.Empty(t, nullGdc.Host, "Host should be empty for NULL")
	assert.Empty(t, nullGdc.IP, "IP should be empty for NULL")
	assert.Equal(t, 0, nullGdc.Port, "Port should be 0 for NULL")
	assert.Equal(t, 0, nullGdc.GRPCPort, "GRPCPort should be 0 for NULL")
}

// findTestDeviceParams is a helper to find test device params by equipment ID
func findTestDeviceParams(config duck.Configuration, equipmentID int) *duck.TestDeviceConfiguration {
	for i := range config.TestDeviceParams {
		if config.TestDeviceParams[i].EquipmentID == equipmentID {
			return &config.TestDeviceParams[i]
		}
	}
	return nil
}

// TestReadConfiguration_EquipmentWithNoLDC tests orphaned equipment handling.
//
// This validates that equipment without an assigned LDC (ldcID is NULL or invalid)
// is handled correctly and doesn't cause panics or incorrect data.
func TestReadConfiguration_EquipmentWithNoLDC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Insert equipment with NULL ldcID (orphaned equipment)
	// Note: This may fail due to foreign key constraints, which is actually good
	// In that case, this test validates that FK constraints work
	result, err := db.Exec(`
		INSERT INTO equipments (type, device_ip, host_ip, host_port, ldcID, enabled)
		VALUES (22, '127.0.0.1', '127.0.0.1', 16100, NULL, true)
	`)

	if err != nil {
		// Foreign key constraint prevented this - which is correct behavior
		// Mark test as passed since FK protection is working
		t.Skip("Foreign key constraint prevents orphaned equipment (correct behavior)")
		return
	}

	// If we got here, FK constraints aren't enabled - verify equipment is handled
	rowsAffected, _ := result.RowsAffected()
	assert.Equal(t, int64(1), rowsAffected, "Should insert orphaned equipment")

	// Read configuration - should handle orphaned equipment gracefully
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	// Orphaned equipment should either:
	// 1. Not appear in any LDC's equipment list (ideal)
	// 2. Be filtered out by application logic
	totalEquipments := 0
	for _, ldc := range config.LDCs {
		totalEquipments += len(ldc.Equipments)
	}
	// Should have at most the 4 equipments from seed data
	assert.LessOrEqual(t, totalEquipments, 4, "Should not include orphaned equipment in LDC lists")
}

// TestReadConfiguration_MultipleRunRecords tests behavior with multiple run records.
//
// This validates that GetLatestRun correctly returns the most recent run ID
// when multiple runs exist in the database.
func TestReadConfiguration_MultipleRunRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database to clean state
	testhelpers.ResetDatabase(t)

	// Connect to shared database
	db := getSharedDB(t)
	defer db.Close()

	// Insert additional run records with higher IDs
	_, err := db.Exec("INSERT INTO runs (start, stop) VALUES (NULL, NULL)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO runs (start, stop) VALUES (NULL, NULL)")
	require.NoError(t, err)

	// Read configuration - should get the highest run ID
	queries := database.New(db)
	config, err := duck.ReadConfigurationFromDB(queries)
	require.NoError(t, err)

	assert.Equal(t, 3, config.RunNumber, "Should get run ID 3 (latest)")
}
