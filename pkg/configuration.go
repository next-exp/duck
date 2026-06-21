package duck

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"gopkg.in/yaml.v3"
)

// DatabaseOpener defines the interface for opening database connections.
// This allows for mocking in tests without requiring actual database connections.
type DatabaseOpener interface {
	Open(driverName, dataSourceName string) (*sql.DB, error)
}

// Sleeper defines the interface for sleeping.
// This allows for mocking time.Sleep in tests to avoid actual delays.
type Sleeper interface {
	Sleep(duration time.Duration)
}

// defaultDatabaseOpener is the real implementation that uses sql.Open
type defaultDatabaseOpener struct{}

func (d defaultDatabaseOpener) Open(driverName, dataSourceName string) (*sql.DB, error) {
	return sql.Open(driverName, dataSourceName)
}

// defaultSleeper is the real implementation that uses time.Sleep
type defaultSleeper struct{}

func (d defaultSleeper) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

type ConfigurationFile struct {
	Database    DatabaseConfiguration    `yaml:"database"`
	Centrifugal CentrifugalConfiguration `yaml:"centrifugal"`
	LogLevel    slog.Level               `yaml:"log_level"`
	GoStats     bool                     `yaml:"go_stats"`
}

type DatabaseConfiguration struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string `yaml:"hostname"`
	Port     string `yaml:"port"`
	Database string `yaml:"dbname"`
}

type CentrifugalConfiguration struct {
	Host  string `yaml:"host"`
	Port  int    `yaml:"port"`
	Token string `yaml:"token"`
}

func ReadConfigurationFile(filename string) (ConfigurationFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ConfigurationFile{}, err
	}

	var configuration ConfigurationFile
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		return ConfigurationFile{}, err
	}
	return configuration, nil
}

func BuildDatabaseURI(dbConf DatabaseConfiguration) string {
	return fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", dbConf.Username,
		dbConf.Password, dbConf.Hostname, dbConf.Port, dbConf.Database)
}

// OpenDatabaseConnection opens a database connection using the provided DatabaseOpener.
// If opener is nil, it uses the default opener (sql.Open).
func OpenDatabaseConnection(dbURI string, opener DatabaseOpener) (*sql.DB, error) {
	if opener == nil {
		opener = defaultDatabaseOpener{}
	}
	return opener.Open("mysql", dbURI)
}

func ParseDatabaseConfiguration(dbConf DatabaseConfiguration) (*sql.DB, error) {
	dbURI := BuildDatabaseURI(dbConf)
	db, err := OpenDatabaseConnection(dbURI, nil)
	if err != nil {
		return nil, err
	}

	// Test the connection to ensure database is selected
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ParseDatabaseConfigurationWithRetry attempts to connect to the database with exponential backoff retry logic.
// This is useful in containerized environments where the database may not be immediately available on startup.
// If opener or sleeper are nil, default implementations are used.
func ParseDatabaseConfigurationWithRetry(dbConf DatabaseConfiguration, maxRetries int, initialDelay time.Duration, opener DatabaseOpener, sleeper Sleeper) (*sql.DB, error) {
	if opener == nil {
		opener = defaultDatabaseOpener{}
	}
	if sleeper == nil {
		sleeper = defaultSleeper{}
	}

	dbURI := BuildDatabaseURI(dbConf)
	delay := initialDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err := OpenDatabaseConnection(dbURI, opener)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to open database connection after %d attempts: %w", maxRetries, err)
			}
			log.Printf("[WARN] Failed to open database connection (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, delay)
			sleeper.Sleep(delay)
			delay *= 2 // Exponential backoff
			continue
		}

		// Test the connection with a ping
		err = db.Ping()
		if err != nil {
			db.Close() // Close the connection before retrying
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
			}
			log.Printf("[WARN] Failed to ping database (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, err, delay)
			sleeper.Sleep(delay)
			delay *= 2 // Exponential backoff
			continue
		}

		// Connection successful
		if attempt > 1 {
			log.Printf("[INFO] Successfully connected to database after %d attempts", attempt)
		}
		return db, nil
	}
	return nil, fmt.Errorf("failed to connect to database after %d attempts", maxRetries)
}

// ReadConfiguration reads the configuration from a YAML file and the database.
// If opener or sleeper are nil, default implementations are used.
func ReadConfiguration(filename string, opener DatabaseOpener, sleeper Sleeper) (Configuration, error) {
	confFile, err := ReadConfigurationFile(filename)
	if err != nil {
		return Configuration{}, err
	}

	// Use retry logic with 10 retries and 1 second initial delay
	// Total max wait time: ~63 seconds (1+2+4+8+16+32)
	db, err := ParseDatabaseConfigurationWithRetry(confFile.Database, 10, 1*time.Second, opener, sleeper)
	if err != nil {
		return Configuration{}, err
	}

	queries := database.New(db)
	configuration, err := ReadConfigurationFromDB(queries)
	if err != nil {
		return Configuration{}, err
	}

	err = db.Close()
	if err != nil {
		return Configuration{}, err
	}

	return configuration, err
}

// ReadConfigurationWithoutRetry reads configuration from YAML and database without retry logic.
// This is useful for tests where the database is already known to be ready (e.g., with dockertest).
// Using this in tests avoids unnecessary 60+ second delays.
func ReadConfigurationWithoutRetry(filename string) (Configuration, error) {
	confFile, err := ReadConfigurationFile(filename)
	if err != nil {
		return Configuration{}, err
	}

	// Parse database configuration directly without retries
	dbURI := BuildDatabaseURI(confFile.Database)
	db, err := OpenDatabaseConnection(dbURI, nil)
	if err != nil {
		return Configuration{}, err
	}

	// Test the connection to ensure database is selected
	err = db.Ping()
	if err != nil {
		db.Close()
		return Configuration{}, fmt.Errorf("failed to ping database: %w", err)
	}

	queries := database.New(db)
	configuration, err := ReadConfigurationFromDB(queries)
	if err != nil {
		db.Close()
		return Configuration{}, err
	}

	err = db.Close()
	if err != nil {
		return Configuration{}, err
	}

	return configuration, nil
}
