package testhelpers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// Shared MySQL container for database tests
var (
	sharedMySQLPool     *dockertest.Pool
	sharedMySQLResource *dockertest.Resource
	sharedMySQLPort     string
)

// SetupSharedMySQL creates a shared MySQL container for all tests in a package
// Call this from TestMain() before running tests
func SetupSharedMySQL() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(fmt.Sprintf("Could not construct pool: %v", err))
	}

	err = pool.Client.Ping()
	if err != nil {
		panic(fmt.Sprintf("Could not connect to Docker: %v", err))
	}

	resource, err := pool.RunWithOptions(getMySQLOptions(), func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		config.PublishAllPorts = true
	})
	if err != nil {
		panic(fmt.Sprintf("Could not start MySQL container: %v", err))
	}

	// Don't set expiration - we'll manage lifecycle manually

	// Wait for MySQL to be ready
	pool.MaxWait = 120 * time.Second
	err = pool.Retry(func() error {
		host := getContainerHost(resource)
		port := "3306"
		if host == "127.0.0.1" {
			port = resource.GetPort("3306/tcp")
		}

		db, err := sql.Open("mysql", fmt.Sprintf("root:testpass@(%s:%s)/duck_test?parseTime=true", host, port))
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	})
	if err != nil {
		panic(fmt.Sprintf("MySQL did not become ready in time: %v", err))
	}

	// Store shared resources
	sharedMySQLPool = pool
	sharedMySQLResource = resource
	if getContainerHost(resource) == "127.0.0.1" {
		sharedMySQLPort = resource.GetPort("3306/tcp")
	} else {
		sharedMySQLPort = "3306"
	}

	fmt.Printf("Shared MySQL container ready: 127.0.0.1:%s\n", sharedMySQLPort)

	// Load schema and initial seed data
	loadSchemaAndSeed()
}

// TeardownSharedMySQL stops the shared MySQL container
// Call this from TestMain() after running tests
func TeardownSharedMySQL() {
	if sharedMySQLResource != nil {
		fmt.Println("Stopping shared MySQL container...")
		if err := sharedMySQLPool.Purge(sharedMySQLResource); err != nil {
			fmt.Printf("Failed to purge MySQL container: %v\n", err)
		}
		sharedMySQLPool = nil
		sharedMySQLResource = nil
		sharedMySQLPort = ""
	}
}

// GetSharedDBConfig returns the database configuration for the shared MySQL
func GetSharedDBConfig() duck.DatabaseConfiguration {
	if sharedMySQLResource == nil {
		panic("Shared MySQL not initialized. Call SetupSharedMySQL() first from TestMain()")
	}
	return duck.DatabaseConfiguration{
		Username: "root",
		Password: "testpass",
		Hostname: "127.0.0.1",
		Port:     sharedMySQLPort,
		Database: "duck_test",
	}
}

// ResetDatabase resets the database to initial state by truncating tables and re-seeding
// Call this at the beginning of each test to ensure clean state
func ResetDatabase(t *testing.T) {
	if sharedMySQLResource == nil {
		panic("Shared MySQL not initialized. Call SetupSharedMySQL() first from TestMain()")
	}

	// Connect to database
	connStr := GetMySQLConnectionString(sharedMySQLResource)
	db, err := sql.Open("mysql", connStr)
	require.NoError(t, err, "Could not connect to database for reset")
	defer db.Close()

	// List of tables to truncate (in order to avoid foreign key constraints)
	tables := []string{
		"testDeviceParams",
		"equipments",
		"ldcs",
		"gdcs",
		"decoderParams",
		"duckParams",
		"runs",
		"errors",
		"events",
		"data",
		"rates",
	}

	// Disable foreign key checks
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS=0")
	require.NoError(t, err, "Failed to disable foreign key checks")

	// Truncate all tables
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		// Don't fail if table doesn't exist
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}

	// Re-enable foreign key checks
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS=1")
	require.NoError(t, err, "Failed to enable foreign key checks")

	// Re-seed the database
	seedFile := filepath.Join(GetProjectRoot(), "integration_tests", "fixtures", "database", "integration_seed.sql")
	executeSQLFileDirect(t, sharedMySQLResource, seedFile)

	t.Log("Database reset complete")
}

// loadSchemaAndSeed loads schema and seed data into shared database
func loadSchemaAndSeed() {
	fmt.Println("Loading schema and seed data into shared MySQL...")

	// Load schema
	schemaFile := filepath.Join(GetProjectRoot(), "database", "schema.sql")
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		panic(fmt.Sprintf("Could not read schema file: %v", err))
	}

	// Connect to database
	connStr := GetMySQLConnectionString(sharedMySQLResource)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to database: %v", err))
	}
	defer db.Close()

	// Execute schema
	statements := splitSQLStatements(fixSQLForTestDB(string(content)))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		_, err := db.Exec(stmt)
		if err != nil {
			panic(fmt.Sprintf("Failed to execute schema statement: %s\nError: %v", stmt, err))
		}
	}

	// Load seed data
	seedFile := filepath.Join(GetProjectRoot(), "integration_tests", "fixtures", "database", "integration_seed.sql")
	seedContent, err := os.ReadFile(seedFile)
	if err != nil {
		panic(fmt.Sprintf("Could not read seed file: %v", err))
	}

	statements = splitSQLStatements(string(seedContent))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		_, err := db.Exec(stmt)
		if err != nil {
			panic(fmt.Sprintf("Failed to execute seed statement: %s\nError: %v", stmt, err))
		}
	}

	fmt.Printf("Schema and seed data loaded (%d schema statements, %d seed statements)\n",
		len(splitSQLStatements(fixSQLForTestDB(string(content)))), len(statements))
}

// GetMySQLConnectionString returns connection string for test MySQL
func GetMySQLConnectionString(resource *dockertest.Resource) string {
	host := getContainerHost(resource)
	port := "3306"
	if host == "127.0.0.1" {
		port = resource.GetPort("3306/tcp")
	}
	return fmt.Sprintf("root:testpass@(%s:%s)/duck_test?parseTime=true", host, port)
}

// executeSQLFileDirect executes SQL using Go MySQL driver directly
func executeSQLFileDirect(t *testing.T, resource *dockertest.Resource, sqlFile string) {
	// Read SQL file
	content, err := os.ReadFile(sqlFile)
	require.NoError(t, err, "Could not read SQL file %s", sqlFile)

	// Fix SQL content for test database
	fixedContent := fixSQLForTestDB(string(content))

	t.Logf("Executing SQL file %s directly (%d bytes)", sqlFile, len(fixedContent))

	// Connect to database
	connStr := GetMySQLConnectionString(resource)
	db, err := sql.Open("mysql", connStr)
	require.NoError(t, err, "Could not connect to database for SQL execution")
	defer db.Close()

	// Split SQL into individual statements and execute them
	statements := splitSQLStatements(fixedContent)
	t.Logf("Found %d statements in %s", len(statements), sqlFile)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		_, err := db.Exec(stmt)
		require.NoError(t, err, "Failed to execute SQL statement: %s", stmt)
	}

	t.Logf("Successfully executed %d SQL statements from %s", len(statements), sqlFile)
}

// splitSQLStatements splits SQL content into individual statements
func splitSQLStatements(sqlContent string) []string {
	// Simple split by semicolon, handling the fact that MySQL statements end with ;
	statements := strings.Split(sqlContent, ";")
	var result []string

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)

		// Skip empty statements
		if stmt == "" {
			continue
		}

		// Remove comment lines while preserving SQL
		lines := strings.Split(stmt, "\n")
		var sqlLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip comment lines
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			if trimmed != "" {
				sqlLines = append(sqlLines, trimmed)
			}
		}

		// Join the remaining SQL lines
		cleanStmt := strings.Join(sqlLines, " ")

		if cleanStmt != "" {
			result = append(result, cleanStmt)
		}
	}

	return result
}

// fixSQLForTestDB modifies SQL content for the test database
func fixSQLForTestDB(sqlContent string) string {
	// Replace database name references line by line to avoid malformed SQL
	lines := strings.Split(sqlContent, "\n")
	var fixedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip problematic lines entirely
		if strings.HasPrefix(trimmed, "CREATE DATABASE duck;") ||
			strings.HasPrefix(trimmed, "GRANT ALL ON duck.*") {
			// Skip these lines entirely
			continue
		} else if strings.HasPrefix(trimmed, "use duck;") {
			fixedLines = append(fixedLines, "USE duck_test;")
		} else {
			// Keep the line as-is (including comments and empty lines)
			fixedLines = append(fixedLines, line)
		}
	}

	return strings.Join(fixedLines, "\n")
}

type RunRecord struct {
	ID    int
	Start *time.Time
	Stop  *time.Time
}

type ComponentStats struct {
	ID     int
	Events int64
	Bytes  int64
	Errors int64
}

type RunStatistics struct {
	Run      RunRecord
	LDCStats map[int]ComponentStats
	GDCStats map[int]ComponentStats
}

func GetRunRecord(t *testing.T, runID int) RunRecord {
	dbConfig := GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	var run RunRecord
	run.ID = runID

	var start, stop sql.NullTime
	err = db.QueryRow("SELECT id, start, stop FROM runs WHERE id = ?", runID).Scan(&run.ID, &start, &stop)
	require.NoError(t, err, "Failed to query run record")

	if start.Valid {
		run.Start = &start.Time
	}
	if stop.Valid {
		run.Stop = &stop.Time
	}

	return run
}

func GetRunStatistics(t *testing.T, runID int) RunStatistics {
	dbConfig := GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	stats := RunStatistics{
		Run:      GetRunRecord(t, runID),
		LDCStats: make(map[int]ComponentStats),
		GDCStats: make(map[int]ComponentStats),
	}

	ldcEventsRows, err := db.Query("SELECT ldc_id, events FROM events WHERE run = ? AND ldc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query LDC events")
	for ldcEventsRows.Next() {
		var ldcID int
		var events int64
		require.NoError(t, ldcEventsRows.Scan(&ldcID, &events))
		s := stats.LDCStats[ldcID]
		s.ID = ldcID
		s.Events = events
		stats.LDCStats[ldcID] = s
	}
	ldcEventsRows.Close()

	ldcBytesRows, err := db.Query("SELECT ldc_id, bytes FROM data WHERE run = ? AND ldc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query LDC bytes")
	for ldcBytesRows.Next() {
		var ldcID int
		var bytes int64
		require.NoError(t, ldcBytesRows.Scan(&ldcID, &bytes))
		s := stats.LDCStats[ldcID]
		s.ID = ldcID
		s.Bytes = bytes
		stats.LDCStats[ldcID] = s
	}
	ldcBytesRows.Close()

	ldcErrorsRows, err := db.Query("SELECT ldc_id, errors FROM errors WHERE run = ? AND ldc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query LDC errors")
	for ldcErrorsRows.Next() {
		var ldcID int
		var errors int64
		require.NoError(t, ldcErrorsRows.Scan(&ldcID, &errors))
		s := stats.LDCStats[ldcID]
		s.ID = ldcID
		s.Errors = errors
		stats.LDCStats[ldcID] = s
	}
	ldcErrorsRows.Close()

	gdcEventsRows, err := db.Query("SELECT gdc_id, events FROM events WHERE run = ? AND gdc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query GDC events")
	for gdcEventsRows.Next() {
		var gdcID int
		var events int64
		require.NoError(t, gdcEventsRows.Scan(&gdcID, &events))
		s := stats.GDCStats[gdcID]
		s.ID = gdcID
		s.Events = events
		stats.GDCStats[gdcID] = s
	}
	gdcEventsRows.Close()

	gdcBytesRows, err := db.Query("SELECT gdc_id, bytes FROM data WHERE run = ? AND gdc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query GDC bytes")
	for gdcBytesRows.Next() {
		var gdcID int
		var bytes int64
		require.NoError(t, gdcBytesRows.Scan(&gdcID, &bytes))
		s := stats.GDCStats[gdcID]
		s.ID = gdcID
		s.Bytes = bytes
		stats.GDCStats[gdcID] = s
	}
	gdcBytesRows.Close()

	gdcErrorsRows, err := db.Query("SELECT gdc_id, errors FROM errors WHERE run = ? AND gdc_id IS NOT NULL", runID)
	require.NoError(t, err, "Failed to query GDC errors")
	for gdcErrorsRows.Next() {
		var gdcID int
		var errors int64
		require.NoError(t, gdcErrorsRows.Scan(&gdcID, &errors))
		s := stats.GDCStats[gdcID]
		s.ID = gdcID
		s.Errors = errors
		stats.GDCStats[gdcID] = s
	}
	gdcErrorsRows.Close()

	return stats
}

func VerifyRunTimestamps(t *testing.T, run RunRecord) {
	require.NotNil(t, run.Start, "Run start time should be set")
	require.NotNil(t, run.Stop, "Run stop time should be set")
	t.Logf("Run %d: start=%v, stop=%v", run.ID, run.Start, run.Stop)
}

func VerifyRunTimestampsSetWithoutStats(t *testing.T, runID int) {
	run := GetRunRecord(t, runID)
	require.NotNil(t, run.Start, "Run start time should be set (forceStop)")
	require.NotNil(t, run.Stop, "Run stop time should be set (forceStop)")
	t.Logf("Run %d (forceStop): start=%v, stop=%v", run.ID, run.Start, run.Stop)

	dbConfig := GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	var eventCount, dataCount, errorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM events WHERE run = ?", runID).Scan(&eventCount)
	require.NoError(t, err)
	require.Equal(t, 0, eventCount, "Events should NOT be stored for forceStop")

	err = db.QueryRow("SELECT COUNT(*) FROM data WHERE run = ?", runID).Scan(&dataCount)
	require.NoError(t, err)
	require.Equal(t, 0, dataCount, "Data should NOT be stored for forceStop")

	err = db.QueryRow("SELECT COUNT(*) FROM errors WHERE run = ?", runID).Scan(&errorCount)
	require.NoError(t, err)
	require.Equal(t, 0, errorCount, "Errors should NOT be stored for forceStop")

	t.Logf("Run %d: confirmed no statistics stored (events=%d, data=%d, errors=%d)", runID, eventCount, dataCount, errorCount)
}
