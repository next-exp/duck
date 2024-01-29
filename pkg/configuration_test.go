package duck

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockDatabaseOpener is a mock implementation of DatabaseOpener for testing.
type mockDatabaseOpener struct {
	openFunc func(driverName, dataSourceName string) (*sql.DB, error)
}

func (m *mockDatabaseOpener) Open(driverName, dataSourceName string) (*sql.DB, error) {
	if m.openFunc != nil {
		return m.openFunc(driverName, dataSourceName)
	}
	return nil, fmt.Errorf("mockDatabaseOpener: no open function set")
}

// mockSleeper is a mock implementation of Sleeper for testing.
// It tracks sleep calls without actually sleeping.
type mockSleeper struct {
	mu          sync.Mutex
	sleepCalls []time.Duration
}

func (m *mockSleeper) Sleep(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sleepCalls = append(m.sleepCalls, duration)
}

// GetSleepCalls returns a copy of the sleep calls made.
func (m *mockSleeper) GetSleepCalls() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]time.Duration, len(m.sleepCalls))
	copy(calls, m.sleepCalls)
	return calls
}

// SleepCount returns the number of times Sleep was called.
func (m *mockSleeper) SleepCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sleepCalls)
}

// Helper functions to create test database models

func createTestGdc(id int32, name string, enabled bool) database.Gdc {
	return database.Gdc{
		ID:             id,
		Name:           sql.NullString{String: name, Valid: true},
		Hostname:       sql.NullString{String: "host" + name, Valid: true},
		Ip:             sql.NullString{String: "192.168.1." + fmt.Sprint(id), Valid: true},
		Port:           sql.NullInt32{Int32: 6000 + id, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: 50000 + id, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: 12000 + id, Valid: true},
		Datapath:       sql.NullString{String: "/data", Valid: true},
		Enabled:        sql.NullBool{Bool: enabled, Valid: true},
		Writeoutput:    sql.NullBool{Bool: true, Valid: true},
		Decode:         sql.NullBool{Bool: false, Valid: true},
	}
}

func createTestLdc(id int32, name string, enabled bool) database.Ldc {
	return database.Ldc{
		ID:             id,
		Name:           sql.NullString{String: name, Valid: true},
		Hostname:       sql.NullString{String: "ldchost" + name, Valid: true},
		Ip:             sql.NullString{String: "192.168.2." + fmt.Sprint(id), Valid: true},
		GrpcPort:       sql.NullInt32{Int32: 50000 + id, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: 12000 + id, Valid: true},
		Enabled:        sql.NullBool{Bool: enabled, Valid: true},
	}
}

func createTestEquipment(id int32, ldcID int32, enabled bool) database.Equipment {
	return database.Equipment{
		ID:       id,
		Type:     sql.NullInt32{Int32: 20 + int32(id), Valid: true},
		DeviceIp: sql.NullString{String: "192.168.3." + fmt.Sprint(id), Valid: true},
		HostIp:   sql.NullString{String: "192.168.3." + fmt.Sprint(id+10), Valid: true},
		HostPort: sql.NullInt32{Int32: 6000 + int32(id), Valid: true},
		Enabled:  sql.NullBool{Bool: enabled, Valid: true},
		Ldcid:    sql.NullInt32{Int32: ldcID, Valid: true},
	}
}

func createTestDuckParam() database.Duckparam {
	return database.Duckparam{
		Filesize:         1024,
		Experiment:       "NEXT100",
		Packetsize:       1500,
		Npacketsinbuffer: 1000,
		Buffertimeout:    100,
		Equipmentdatach:  10,
		Receivech:        20,
		Datach:           30,
		Metricsch:        40,
		Tcpconnectionsch: 50,
		Decoderch:        70,
		Decoderworkers:   5,
		Writerch:         60,
	}
}

func createTestDecoderParam() database.Decoderparam {
	return database.Decoderparam{
		ExtTrigger:       1,
		TrgCode1:         10,
		TrgCode2:         20,
		ReadPmts:         sql.NullBool{Bool: true, Valid: true},
		ReadSipms:        sql.NullBool{Bool: false, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: true, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: false, Valid: true},
		NoDb:             sql.NullBool{Bool: false, Valid: true},
		Discard:          sql.NullBool{Bool: false, Valid: true},
		Host:             "localhost",
		User:             "decoder_user",
		Passwd:           "decoder_pass",
		DbName:           "decoder_db",
		WriteData:        sql.NullBool{Bool: true, Valid: true},
		UseBlosc:         sql.NullBool{Bool: true, Valid: true},
		BloscAlgorithm:   "lz4",
		CompressionLevel: 5,
		BitShuffle:       "yes",
	}
}

func TestReadConfigurationFile_ValidYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ConfigurationFile
		wantErr bool
	}{
		{
			name: "valid complete configuration",
			input: `database:
  username: "testuser"
  password: "testpass"
  hostname: "localhost"
  port: "3306"
  dbname: "testdb"
centrifugal:
  host: "localhost"
  port: 8000
  token: "test-token"`,
			want: ConfigurationFile{
				Database: DatabaseConfiguration{
					Username: "testuser",
					Password: "testpass",
					Hostname: "localhost",
					Port:     "3306",
					Database: "testdb",
				},
				Centrifugal: CentrifugalConfiguration{
					Host:  "localhost",
					Port:  8000,
					Token: "test-token",
				},
				LogLevel: slog.LevelInfo, // Default value
			},
			wantErr: false,
		},
		{
			name: "valid with different port",
			input: `database:
  username: "user"
  password: "pass"
  hostname: "host"
  port: "3307"
  dbname: "db"
centrifugal:
  host: "host"
  port: 8123
  token: "token"`,
			want: ConfigurationFile{
				Database: DatabaseConfiguration{
					Username: "user",
					Password: "pass",
					Hostname: "host",
					Port:     "3307",
					Database: "db",
				},
				Centrifugal: CentrifugalConfiguration{
					Host:  "host",
					Port:  8123,
					Token: "token",
				},
				LogLevel: slog.LevelInfo, // Default value
			},
			wantErr: false,
		},
		{
			name: "missing database section",
			input: `centrifugal:
  host: "localhost"
  port: 8000
  token: "token"`,
			wantErr: false, // YAML will parse but fields will be empty
		},
		{
			name:    "invalid yaml",
			input:   "invalid: yaml: content: [}",
			wantErr: true,
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: false, // Empty YAML is valid, just creates zero values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpfile := filepath.Join(t.TempDir(), "config.yml")
			err := os.WriteFile(tmpfile, []byte(tt.input), 0644)
			require.NoError(t, err)

			// Test the function
			got, err := ReadConfigurationFile(tmpfile)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.name != "empty file" && tt.name != "missing database section" {
				assert.Equal(t, tt.want.Database.Username, got.Database.Username)
				assert.Equal(t, tt.want.Database.Password, got.Database.Password)
				assert.Equal(t, tt.want.Database.Hostname, got.Database.Hostname)
				assert.Equal(t, tt.want.Database.Port, got.Database.Port)
				assert.Equal(t, tt.want.Database.Database, got.Database.Database)
				assert.Equal(t, tt.want.Centrifugal.Host, got.Centrifugal.Host)
				assert.Equal(t, tt.want.Centrifugal.Port, got.Centrifugal.Port)
				assert.Equal(t, tt.want.Centrifugal.Token, got.Centrifugal.Token)
				// LogLevel is compared with default value
			}
		})
	}
}

func TestReadConfigurationFile_FileNotFound(t *testing.T) {
	_, err := ReadConfigurationFile("/nonexistent/path/to/file.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestReadConfigurationFromDB_Success tests successful configuration reading from database
func TestReadConfigurationFromDB_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	// Setup expectations
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{
			createTestGdc(1, "gdc1", true),
			createTestGdc(2, "gdc2", false),
		}, nil)

	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{
			createTestLdc(1, "ldc1", true),
			createTestLdc(2, "ldc2", false),
		}, nil)

	mockQuerier.On("ListEquipments", mock.Anything).
		Return([]database.Equipment{
			createTestEquipment(1, 1, true),
			createTestEquipment(2, 1, false),
			createTestEquipment(3, 2, true),
		}, nil)

	mockQuerier.On("GetLatestRun", mock.Anything).
		Return(int32(13318), nil)

	mockQuerier.On("GetDuckParams", mock.Anything).
		Return(createTestDuckParam(), nil)

	mockQuerier.On("GetDecoderParams", mock.Anything).
		Return(createTestDecoderParam(), nil)

	mockQuerier.On("GetTestDeviceParams", mock.Anything).
		Return([]database.Testdeviceparam{
			{
				EquipmentID:        1,
				EventRateMs:        12,
				PacketsPerEvent:    85,
				PacketSize:         500,
				ErrorInjectionRate: 0.0,
				MaxEvents:          0,
			},
		}, nil)

	// Execute
	config, err := ReadConfigurationFromDB(mockQuerier)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 13318, config.RunNumber)
	assert.Len(t, config.GDCs, 2)
	assert.Len(t, config.LDCs, 2)

	// Check GDCs
	assert.Equal(t, 1, config.GDCs[0].ID)
	assert.Equal(t, "gdc1", config.GDCs[0].Name)
	assert.Equal(t, "hostgdc1", config.GDCs[0].Host)
	assert.Equal(t, "192.168.1.1", config.GDCs[0].IP)
	assert.Equal(t, 6001, config.GDCs[0].Port)
	assert.True(t, config.GDCs[0].Enabled)
	assert.True(t, config.GDCs[0].WriteOutput)
	assert.False(t, config.GDCs[0].Decode)

	// Check LDCs and equipment assignment
	assert.Equal(t, 1, config.LDCs[0].ID)
	assert.Equal(t, "ldc1", config.LDCs[0].Name)
	assert.Len(t, config.LDCs[0].Equipments, 2)
	assert.Equal(t, 1, config.LDCs[0].Equipments[0].ID)
	assert.Equal(t, 2, config.LDCs[0].Equipments[1].ID)

	assert.Equal(t, 2, config.LDCs[1].ID)
	assert.Len(t, config.LDCs[1].Equipments, 1)
	assert.Equal(t, 3, config.LDCs[1].Equipments[0].ID)

	// Check Duck configuration
	assert.Equal(t, 1024, config.Duck.Filesize)
	assert.Equal(t, "NEXT100", config.Duck.Experiment)
	assert.Equal(t, 1500, config.Duck.PacketSize)
	assert.Equal(t, 5, config.Duck.DecoderWorkers)

	// Check Decoder configuration
	assert.Equal(t, 1, config.Decoder.ExtTrigger)
	assert.Equal(t, 10, config.Decoder.TrgCode1)
	assert.Equal(t, 20, config.Decoder.TrgCode2)
	assert.True(t, config.Decoder.ReadPMTs)
	assert.False(t, config.Decoder.ReadSiPMs)
	assert.True(t, config.Decoder.WriteData)
}

// TestReadConfigurationFromDB_DatabaseError tests database error handling
func TestReadConfigurationFromDB_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	// First query fails
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, fmt.Errorf("database connection failed"))

	config, err := ReadConfigurationFromDB(mockQuerier)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.Equal(t, Configuration{}, config)
}

// TestReadConfigurationFromDB_MissingData tests handling of missing/empty data
func TestReadConfigurationFromDB_MissingData(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	// Empty results
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).
		Return([]database.Equipment{}, nil)

	// Run query fails
	mockQuerier.On("GetLatestRun", mock.Anything).
		Return(int32(0), fmt.Errorf("no rows in result set"))

	config, err := ReadConfigurationFromDB(mockQuerier)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no rows in result set")
	assert.Equal(t, Configuration{}, config)
}

// TestReadConfigurationFromDB_PartialFailure tests partial data reading failures
func TestReadConfigurationFromDB_PartialFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{createTestGdc(1, "gdc1", true)}, nil)

	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{createTestLdc(1, "ldc1", true)}, nil)

	// Equipments query fails
	mockQuerier.On("ListEquipments", mock.Anything).
		Return([]database.Equipment{}, fmt.Errorf("table equipment not accessible"))

	config, err := ReadConfigurationFromDB(mockQuerier)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "table equipment not accessible")
	assert.Equal(t, Configuration{}, config)
}

func TestBuildDatabaseURI(t *testing.T) {
	tests := []struct {
		name   string
		config DatabaseConfiguration
		want   string
	}{
		{
			name: "standard configuration",
			config: DatabaseConfiguration{
				Username: "testuser",
				Password: "testpass",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "testuser:testpass@(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "different host and port",
			config: DatabaseConfiguration{
				Username: "user",
				Password: "pass",
				Hostname: "db.example.com",
				Port:     "5432",
				Database: "mydb",
			},
			want: "user:pass@(db.example.com:5432)/mydb?parseTime=true",
		},
		{
			name: "empty configuration",
			config: DatabaseConfiguration{
				Username: "",
				Password: "",
				Hostname: "",
				Port:     "",
				Database: "",
			},
			want: ":@(:)/?parseTime=true",
		},
		{
			name: "special characters in password",
			config: DatabaseConfiguration{
				Username: "user",
				Password: "p@ssw0rd!",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user:p@ssw0rd!@(localhost:3306)/testdb?parseTime=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDatabaseURI(tt.config)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseDatabaseConfigurationWithRetry_SuccessFirstAttempt tests successful connection on first attempt
func TestParseDatabaseConfigurationWithRetry_SuccessFirstAttempt(t *testing.T) {
	// Create a mock database that will succeed
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer mockDB.Close()

	// Expect a ping
	mock.ExpectPing()

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return mockDB, nil
		},
	}
	sleeper := &mockSleeper{}

	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "localhost",
		Port:     "3306",
		Database: "testdb",
	}

	db, err := ParseDatabaseConfigurationWithRetry(config, 3, 100*time.Millisecond, opener, sleeper)

	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, 0, sleeper.SleepCount(), "Should not sleep on first attempt success")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestParseDatabaseConfigurationWithRetry_InvalidConfig tests with invalid database config
func TestParseDatabaseConfigurationWithRetry_InvalidConfig(t *testing.T) {
	// No need to skip in short mode anymore - this test is now fast!

	config := DatabaseConfiguration{
		Username: "invalid",
		Password: "invalid",
		Hostname: "nonexistent-host-that-does-not-exist",
		Port:     "3306",
		Database: "testdb",
	}

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	sleeper := &mockSleeper{}

	// Use 3 retries
	db, err := ParseDatabaseConfigurationWithRetry(config, 3, 100*time.Millisecond, opener, sleeper)

	// Should fail after retries
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "3 attempts")
	assert.Equal(t, 2, sleeper.SleepCount(), "Should sleep 2 times before failing on 3rd attempt")
}

// TestParseDatabaseConfigurationWithRetry_ExponentialBackoff tests the delay doubling logic
func TestParseDatabaseConfigurationWithRetry_ExponentialBackoff(t *testing.T) {
	// No need to skip in short mode anymore - this test is now fast!

	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "localhost",
		Port:     "3306",
		Database: "testdb",
	}

	initialDelay := 100 * time.Millisecond
	maxRetries := 4

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	sleeper := &mockSleeper{}

	_, err := ParseDatabaseConfigurationWithRetry(config, maxRetries, initialDelay, opener, sleeper)

	assert.Error(t, err)

	// Verify exponential backoff: 100ms, 200ms, 400ms
	sleepCalls := sleeper.GetSleepCalls()
	assert.Equal(t, 3, len(sleepCalls), "Should have 3 sleep calls for 4 retries")

	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
	}
	assert.Equal(t, expected, sleepCalls, "Sleep durations should follow exponential backoff")
}

// TestParseDatabaseConfigurationWithRetry_AllRetriesFail tests failure after all retries
func TestParseDatabaseConfigurationWithRetry_AllRetriesFail(t *testing.T) {
	// No need to skip in short mode anymore - this test is now fast!

	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "192.0.2.1", // TEST-NET-1, guaranteed to not route
		Port:     "3306",
		Database: "testdb",
	}

	maxRetries := 3

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	sleeper := &mockSleeper{}

	db, err := ParseDatabaseConfigurationWithRetry(config, maxRetries, 100*time.Millisecond, opener, sleeper)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("%d attempts", maxRetries))
	assert.Nil(t, db)
	assert.Equal(t, 2, sleeper.SleepCount(), "Should sleep 2 times before final failure")
}

// TestParseDatabaseConfigurationWithRetry_SingleRetry tests with only 1 retry
func TestParseDatabaseConfigurationWithRetry_SingleRetry(t *testing.T) {
	// No need to skip in short mode anymore - this test is now fast!

	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "192.0.2.1", // TEST-NET-1
		Port:     "3306",
		Database: "testdb",
	}

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	sleeper := &mockSleeper{}

	db, err := ParseDatabaseConfigurationWithRetry(config, 1, 100*time.Millisecond, opener, sleeper)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "1 attempts")
	assert.Nil(t, db)
	assert.Equal(t, 0, sleeper.SleepCount(), "Should not sleep when maxRetries is 1")
}

// TestParseDatabaseConfigurationWithRetry_ErrorMessageFormat tests error message contains expected info
func TestParseDatabaseConfigurationWithRetry_ErrorMessageFormat(t *testing.T) {
	// No need to skip in short mode anymore - this test is now fast!

	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "192.0.2.1",
		Port:     "3306",
		Database: "testdb",
	}

	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	sleeper := &mockSleeper{}

	_, err := ParseDatabaseConfigurationWithRetry(config, 2, 100*time.Millisecond, opener, sleeper)

	assert.Error(t, err)
	// Error should mention the number of attempts
	assert.Contains(t, err.Error(), "2 attempts")
}

// TestParseDatabaseConfigurationWithRetry_PingFailureThenSuccess tests retry on ping failure
func TestParseDatabaseConfigurationWithRetry_PingFailureThenSuccess(t *testing.T) {
	config := DatabaseConfiguration{
		Username: "test",
		Password: "test",
		Hostname: "localhost",
		Port:     "3306",
		Database: "testdb",
	}

	attempt := 0
	opener := &mockDatabaseOpener{
		openFunc: func(driverName, dataSourceName string) (*sql.DB, error) {
			// Create mock DBs that fail ping on first attempt, succeed on second
			mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)

			attempt++
			if attempt == 1 {
				// First attempt: ping fails
				mock.ExpectPing().WillReturnError(fmt.Errorf("ping failed"))
			} else {
				// Second attempt: ping succeeds
				mock.ExpectPing()
			}

			return mockDB, nil
		},
	}
	sleeper := &mockSleeper{}

	db, err := ParseDatabaseConfigurationWithRetry(config, 3, 100*time.Millisecond, opener, sleeper)

	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, 1, sleeper.SleepCount(), "Should sleep once after first ping failure")
	// Verify exponential backoff: 100ms
	sleepCalls := sleeper.GetSleepCalls()
	assert.Equal(t, []time.Duration{100 * time.Millisecond}, sleepCalls)
}

// TestReadConfiguration_Success tests the complete ReadConfiguration flow
// TestReadConfiguration_FileReadError tests error handling when config file cannot be read
func TestReadConfiguration_FileReadError(t *testing.T) {
	// Try to read a non-existent file
	config, err := ReadConfiguration("/nonexistent/path/to/config.yml", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, Configuration{}, config)
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestReadConfiguration_InvalidYAML tests error handling for invalid YAML
func TestReadConfiguration_InvalidYAML(t *testing.T) {
	// Create temporary YAML config file with invalid content
	yamlContent := `database:
  username: "test"
invalid: yaml: content: [}
`
	tmpfile, err := os.CreateTemp("", "config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(yamlContent))
	require.NoError(t, err)
	tmpfile.Close()

	config, err := ReadConfiguration(tmpfile.Name(), nil, nil)

	assert.Error(t, err)
	assert.Equal(t, Configuration{}, config)
}

// TestBuildDatabaseURI_SpecialCharacters tests URI building with special characters
func TestBuildDatabaseURI_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name   string
		config DatabaseConfiguration
		want   string
	}{
		{
			name: "at sign in username",
			config: DatabaseConfiguration{
				Username: "user@domain",
				Password: "pass",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user@domain:pass@(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "colon in password",
			config: DatabaseConfiguration{
				Username: "user",
				Password: "pass:word",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user:pass:word@(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "at sign in password",
			config: DatabaseConfiguration{
				Username: "user",
				Password: "p@ssw0rd",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user:p@ssw0rd@(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "multiple special characters",
			config: DatabaseConfiguration{
				Username: "user@domain",
				Password: "p@ss:w0rd!",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user@domain:p@ss:w0rd!@(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "percent encoding needed",
			config: DatabaseConfiguration{
				Username: "user%name",
				Password: "pass%word",
				Hostname: "localhost",
				Port:     "3306",
				Database: "testdb",
			},
			want: "user%name:pass%word@(localhost:3306)/testdb?parseTime=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDatabaseURI(tt.config)
			// Note: The current implementation doesn't do URL encoding
			// This test documents the current behavior
			assert.Equal(t, tt.want, got)
			// In a production system, you might want to use url.QueryEscape()
			// to properly encode special characters
		})
	}
}

// TestReadConfigurationFile_PartialYAML tests handling of partial YAML configurations
func TestReadConfigurationFile_PartialYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "missing centrifugal section",
			input: `database:
  username: "testuser"
  password: "testpass"
  hostname: "localhost"
  port: "3306"
  dbname: "testdb"
`,
			wantErr: false, // Partial YAML is valid, fields will be empty
		},
		{
			name: "only database section",
			input: `database:
  username: "testuser"
  password: "testpass"
  hostname: "localhost"
  port: "3306"
  dbname: "testdb"
`,
			wantErr: false,
		},
		{
			name: "empty database values",
			input: `database:
  username: ""
  password: ""
  hostname: ""
  port: ""
  dbname: ""
centrifugal:
  host: ""
  port: 0
  token: ""
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile := filepath.Join(t.TempDir(), "config.yml")
			err := os.WriteFile(tmpfile, []byte(tt.input), 0644)
			require.NoError(t, err)

			got, err := ReadConfigurationFile(tmpfile)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}
