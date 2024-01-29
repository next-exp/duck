package testhelpers

import (
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// CreateTestDecoderConfig creates a test decoder configuration with default values
func CreateTestDecoderConfig() duck.DecoderConfiguration {
	return duck.DecoderConfiguration{
		WriteData:        true,
		SplitTrigger:     false,
		ExtTrigger:       1,
		TrgCode1:         1,
		TrgCode2:         2,
		ReadPMTs:         true,
		ReadSiPMs:        true,
		ReadTrigger:      true,
		NoDB:             false,
		Discard:          false,
		Host:             "localhost",
		User:             "test_user",
		Passwd:           "test_password",
		DBName:           "test_db",
		UseBlosc:         true,
		CompressionLevel: 5,
		BloscAlgorithm:   "blosclz",
		BitShuffle:       "bitshuffle",
	}
}

// CreateTestDecoderConfigNoWrite creates a decoder config with WriteData=false
func CreateTestDecoderConfigNoWrite() duck.DecoderConfiguration {
	config := CreateTestDecoderConfig()
	config.WriteData = false
	return config
}

// CreateTestDecoderConfigSplitTrigger creates a decoder config with SplitTrigger=true
func CreateTestDecoderConfigSplitTrigger() duck.DecoderConfiguration {
	config := CreateTestDecoderConfig()
	config.SplitTrigger = true
	config.TrgCode1 = 1
	config.TrgCode2 = 2
	return config
}

// CreateTestDecoderConfigWithBlosc creates a decoder config with specific Blosc settings
func CreateTestDecoderConfigWithBlosc(algorithm string, shuffle string, level int) duck.DecoderConfiguration {
	config := CreateTestDecoderConfig()
	config.BloscAlgorithm = algorithm
	config.BitShuffle = shuffle
	config.CompressionLevel = level
	config.UseBlosc = true
	return config
}

// CreateMinimalDecoderConfig creates a minimal decoder configuration for basic tests
func CreateMinimalDecoderConfig() duck.DecoderConfiguration {
	return duck.DecoderConfiguration{
		WriteData:    false,
		SplitTrigger: false,
		ExtTrigger:   0,
		TrgCode1:     0,
		TrgCode2:     0,
		ReadPMTs:     false,
		ReadSiPMs:    false,
		ReadTrigger:  false,
		NoDB:        true,
		Discard:     false,
		Host:        "",
		User:        "",
		Passwd:      "",
		DBName:      "",
		UseBlosc:    false,
	}
}

// CreateTestDecoderConfigWithDB creates a decoder config with database connection settings
func CreateTestDecoderConfigWithDB(host, user, passwd, dbname string) duck.DecoderConfiguration {
	config := CreateTestDecoderConfig()
	config.Host = host
	config.User = user
	config.Passwd = passwd
	config.DBName = dbname
	config.NoDB = false
	return config
}

// CreateTestDecoderConfigNoDB creates a decoder config without database (NoDB=true)
func CreateTestDecoderConfigNoDB() duck.DecoderConfiguration {
	config := CreateTestDecoderConfig()
	config.NoDB = true
	config.Host = ""
	config.User = ""
	config.Passwd = ""
	config.DBName = ""
	return config
}

// CreateValidGDCEventHeader creates a minimal valid GDC event header for testing
// This is a simplified version for testing purposes
func CreateValidGDCEventHeader(eventID int) []byte {
	// Minimal GDC event header format:
	// [Magic Number (4 bytes)][Event ID (4 bytes)][Data Size (4 bytes)][...payload]
	header := make([]byte, 12)
	// Magic number (example: 0x47444300 = "GDC\0")
	header[0] = 0x47
	header[1] = 0x44
	header[2] = 0x43
	header[3] = 0x00
	// Event ID (little endian)
	header[4] = byte(eventID)
	header[5] = byte(eventID >> 8)
	header[6] = byte(eventID >> 16)
	header[7] = byte(eventID >> 24)
	// Data size (0 for header only)
	header[8] = 0
	header[9] = 0
	header[10] = 0
	header[11] = 0
	return header
}

// CreateValidGDCEvent creates a complete valid GDC event for testing
func CreateValidGDCEvent(eventID int, payloadSize int) []byte {
	header := CreateValidGDCEventHeader(eventID)
	// Update data size in header
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte(i % 256)
	}
	header[8] = byte(payloadSize)
	header[9] = byte(payloadSize >> 8)
	header[10] = byte(payloadSize >> 16)
	header[11] = byte(payloadSize >> 24)

	return append(header, payload...)
}

// CreateInvalidGDCEvent creates an invalid GDC event for error testing
func CreateInvalidGDCEvent() []byte {
	// Invalid magic number
	return []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
}

// CreateEmptyGDCEvent creates an empty GDC event
func CreateEmptyGDCEvent() []byte {
	return []byte{}
}

// CreateMalformedGDCEvent creates a malformed GDC event (truncated header)
func CreateMalformedGDCEvent() []byte {
	return []byte{0x47, 0x44, 0x43} // Only 3 bytes
}
