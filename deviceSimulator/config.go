package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// SimulatorParams holds simulator-specific configuration from database (replay mode)
type SimulatorParams struct {
	EquipmentID int
	FilePath    string
	RateHz      float64 // Absolute event rate in Hz (events per second)
	LoopMode    bool
	MaxEvents   int
	PacketSize  int
}

// GeneratorConfig holds configuration for generate mode
type GeneratorConfig struct {
	EquipmentID        int
	Equipment          duck.Equipment
	RateHz             float64 // Absolute event rate in Hz (events per second)
	PacketsPerEvent    int
	PacketSize         int
	ErrorInjectionRate float64 // 0.0 to 1.0
	MaxEvents          int
}

// LoadEquipmentByID loads equipment configuration from database by ID
func LoadEquipmentByID(db *sql.DB, equipmentID int) (duck.Equipment, error) {
	var equipment duck.Equipment

	query := `
		SELECT id, type, device_ip, host_ip, host_port, ldcID, enabled
		FROM equipments
		WHERE id = ?
	`

	err := db.QueryRow(query, equipmentID).Scan(
		&equipment.ID,
		&equipment.Type,
		&equipment.DeviceIP,
		&equipment.HostIP,
		&equipment.HostPort,
		&equipment.LDC_ID,
		&equipment.Enabled,
	)

	if err != nil {
		return equipment, fmt.Errorf("failed to load equipment %d: %w", equipmentID, err)
	}

	if !equipment.Enabled {
		return equipment, fmt.Errorf("equipment %d is disabled", equipmentID)
	}

	return equipment, nil
}

// LoadSimulatorParams loads simulator parameters from database
func LoadSimulatorParams(db *sql.DB, equipmentID int) (SimulatorParams, error) {
	var params SimulatorParams

	query := `
		SELECT equipmentID, filePath, rateHz, loopMode, maxEvents, packetSize
		FROM simulatorParams
		WHERE equipmentID = ?
	`

	err := db.QueryRow(query, equipmentID).Scan(
		&params.EquipmentID,
		&params.FilePath,
		&params.RateHz,
		&params.LoopMode,
		&params.MaxEvents,
		&params.PacketSize,
	)

	if err == sql.ErrNoRows {
		// Return default parameters if not found
		return SimulatorParams{
			EquipmentID: equipmentID,
			RateHz:      0.1, // Default to 0.1 Hz (1 event per 10 seconds)
			LoopMode:    false,
			MaxEvents:   0,
			PacketSize:  DefaultPacketSize,
		}, nil
	}

	if err != nil {
		return params, fmt.Errorf("failed to load simulator params: %w", err)
	}

	return params, nil
}

// BuildReplayConfig creates a ReplayConfig from database configuration
func BuildReplayConfig(db *sql.DB, equipmentID int, filePath string, rateHz float64, loopMode bool, maxEvents int, fragmentSize int) (ReplayConfig, error) {
	// Load equipment configuration
	equipment, err := LoadEquipmentByID(db, equipmentID)
	if err != nil {
		return ReplayConfig{}, err
	}

	// Override with provided parameters or load from database
	if filePath == "" || rateHz == 0 {
		params, err := LoadSimulatorParams(db, equipmentID)
		if err != nil {
			return ReplayConfig{}, err
		}

		if filePath == "" {
			filePath = params.FilePath
		}
		if rateHz == 0 {
			rateHz = params.RateHz
		}
	}

	// Validate file exists
	if filePath == "" {
		return ReplayConfig{}, fmt.Errorf("file path is required")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return ReplayConfig{}, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Use fragment size, default to 7992 if not provided
	if fragmentSize == 0 {
		fragmentSize = 7992
	}

	return ReplayConfig{
		FilePath:    filePath,
		EquipmentID: uint32(equipmentID),
		Equipment:   equipment,
		RateHz:      rateHz,
		LoopMode:    loopMode,
		MaxEvents:   maxEvents,
		PacketSize:  fragmentSize,
	}, nil
}

// GetDatabaseConnection creates a database connection using environment variables
func GetDatabaseConnection() (*sql.DB, error) {
	// Get database credentials from environment
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}

	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		dbPass = "duck"
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "duck"
	}

	// Build connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// LoadTestDeviceParams loads test device parameters from database for generate mode
func LoadTestDeviceParams(db *sql.DB, equipmentID int) (duck.TestDeviceConfiguration, error) {
	var params duck.TestDeviceConfiguration

	query := `
		SELECT equipment_id, event_rate_ms, packets_per_event, packet_size, error_injection_rate, max_events
		FROM testDeviceParams
		WHERE equipment_id = ?
	`

	err := db.QueryRow(query, equipmentID).Scan(
		&params.EquipmentID,
		&params.EventRateMs,
		&params.PacketsPerEvent,
		&params.PacketSize,
		&params.ErrorInjectionRate,
		&params.MaxEvents,
	)

	if err == sql.ErrNoRows {
		// Return default parameters if not found
		return duck.TestDeviceConfiguration{
			EquipmentID:        equipmentID,
			EventRateMs:        1000, // Default to 1 Hz
			PacketsPerEvent:    10,
			PacketSize:         7992,
			ErrorInjectionRate: 0.0,
			MaxEvents:          0,
		}, nil
	}

	if err != nil {
		return params, fmt.Errorf("failed to load test device params: %w", err)
	}

	return params, nil
}

// BuildGeneratorConfig creates a GeneratorConfig from database configuration and CLI overrides
func BuildGeneratorConfig(db *sql.DB, equipmentID int, rateHz float64, packetsPerEvent int, errorInjectionRate float64, maxEvents int, packetSize int) (GeneratorConfig, error) {
	// Load equipment configuration
	equipment, err := LoadEquipmentByID(db, equipmentID)
	if err != nil {
		return GeneratorConfig{}, err
	}

	// Load test device params from database
	params, err := LoadTestDeviceParams(db, equipmentID)
	if err != nil {
		return GeneratorConfig{}, err
	}

	// Apply CLI overrides (non-zero values override database)
	if rateHz <= 0 {
		// Convert from event_rate_ms to Hz
		if params.EventRateMs > 0 {
			rateHz = 1000.0 / float64(params.EventRateMs)
		} else {
			rateHz = 1.0 // Default 1 Hz
		}
	}
	if packetsPerEvent <= 0 {
		packetsPerEvent = params.PacketsPerEvent
	}
	if errorInjectionRate < 0 {
		errorInjectionRate = params.ErrorInjectionRate
	}
	if maxEvents <= 0 {
		maxEvents = params.MaxEvents
	}
	if packetSize <= 0 {
		packetSize = params.PacketSize
		if packetSize <= 0 {
			packetSize = 7992 // Default NEXT-100 packet size
		}
	}

	return GeneratorConfig{
		EquipmentID:        equipmentID,
		Equipment:          equipment,
		RateHz:             rateHz,
		PacketsPerEvent:    packetsPerEvent,
		PacketSize:         packetSize,
		ErrorInjectionRate: errorInjectionRate,
		MaxEvents:          maxEvents,
	}, nil
}
