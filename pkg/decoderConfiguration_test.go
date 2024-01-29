package duck

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetDecoderParameters_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedDBResult := database.Decoderparam{
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
		User:             "testuser",
		Passwd:           "testpass",
		DbName:           "testdb",
		WriteData:        sql.NullBool{Bool: true, Valid: true},
		UseBlosc:         sql.NullBool{Bool: true, Valid: true},
		BloscAlgorithm:   "lz4",
		CompressionLevel: 5,
		BitShuffle:       "yes",
	}

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(expectedDBResult, nil)

	config, err := getDecoderParameters(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, 1, config.ExtTrigger)
	assert.Equal(t, 10, config.TrgCode1)
	assert.Equal(t, 20, config.TrgCode2)
	assert.True(t, config.ReadPMTs)
	assert.False(t, config.ReadSiPMs)
	assert.True(t, config.ReadTrigger)
	assert.False(t, config.SplitTrigger)
	assert.False(t, config.NoDB)
	assert.False(t, config.Discard)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testpass", config.Passwd)
	assert.Equal(t, "testdb", config.DBName)
	assert.True(t, config.WriteData)
	assert.True(t, config.UseBlosc)
	assert.Equal(t, "lz4", config.BloscAlgorithm)
	assert.Equal(t, 5, config.CompressionLevel)
	assert.Equal(t, "yes", config.BitShuffle)
}

func TestGetDecoderParameters_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedError := errors.New("database connection failed")
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, expectedError)

	config, err := getDecoderParameters(mockQuerier)

	assert.Error(t, err)
	assert.Equal(t, "database connection failed", err.Error())
	assert.Equal(t, DecoderConfiguration{}, config)
}

func TestGetDecoderParameters_NullBoolValues(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	// Test with NullBool where Valid is false (NULL in database)
	dbResult := database.Decoderparam{
		ExtTrigger:       0,
		TrgCode1:         0,
		TrgCode2:         0,
		ReadPmts:         sql.NullBool{Bool: false, Valid: false}, // NULL
		ReadSipms:        sql.NullBool{Bool: false, Valid: false}, // NULL
		ReadTrigger:      sql.NullBool{Bool: false, Valid: false}, // NULL
		SplitTrigger:     sql.NullBool{Bool: false, Valid: false}, // NULL
		NoDb:             sql.NullBool{Bool: false, Valid: false}, // NULL
		Discard:          sql.NullBool{Bool: false, Valid: false}, // NULL
		Host:             "",
		User:             "",
		Passwd:           "",
		DbName:           "",
		WriteData:        sql.NullBool{Bool: false, Valid: false}, // NULL
		UseBlosc:         sql.NullBool{Bool: false, Valid: false}, // NULL
		BloscAlgorithm:   "",
		CompressionLevel: 0,
		BitShuffle:       "",
	}

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(dbResult, nil)

	config, err := getDecoderParameters(mockQuerier)

	require.NoError(t, err)
	// When Valid is false, .Bool returns false
	assert.Equal(t, 0, config.ExtTrigger)
	assert.Equal(t, 0, config.TrgCode1)
	assert.Equal(t, 0, config.TrgCode2)
	assert.False(t, config.ReadPMTs)
	assert.False(t, config.ReadSiPMs)
	assert.False(t, config.ReadTrigger)
	assert.False(t, config.SplitTrigger)
	assert.False(t, config.NoDB)
	assert.False(t, config.Discard)
	assert.Equal(t, "", config.Host)
	assert.Equal(t, "", config.User)
	assert.Equal(t, "", config.Passwd)
	assert.Equal(t, "", config.DBName)
	assert.False(t, config.WriteData)
	assert.False(t, config.UseBlosc)
	assert.Equal(t, "", config.BloscAlgorithm)
	assert.Equal(t, 0, config.CompressionLevel)
	assert.Equal(t, "", config.BitShuffle)
}

func TestGetDecoderParameters_AllBoolsTrue(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	dbResult := database.Decoderparam{
		ExtTrigger:       99,
		TrgCode1:         11,
		TrgCode2:         22,
		ReadPmts:         sql.NullBool{Bool: true, Valid: true},
		ReadSipms:        sql.NullBool{Bool: true, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: true, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: true, Valid: true},
		NoDb:             sql.NullBool{Bool: true, Valid: true},
		Discard:          sql.NullBool{Bool: true, Valid: true},
		Host:             "dbhost",
		User:             "dbuser",
		Passwd:           "dbpass",
		DbName:           "dbname",
		WriteData:        sql.NullBool{Bool: true, Valid: true},
		UseBlosc:         sql.NullBool{Bool: true, Valid: true},
		BloscAlgorithm:   "zstd",
		CompressionLevel: 9,
		BitShuffle:       "no",
	}

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(dbResult, nil)

	config, err := getDecoderParameters(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, 99, config.ExtTrigger)
	assert.Equal(t, 11, config.TrgCode1)
	assert.Equal(t, 22, config.TrgCode2)
	assert.True(t, config.ReadPMTs)
	assert.True(t, config.ReadSiPMs)
	assert.True(t, config.ReadTrigger)
	assert.True(t, config.SplitTrigger)
	assert.True(t, config.NoDB)
	assert.True(t, config.Discard)
	assert.Equal(t, "dbhost", config.Host)
	assert.Equal(t, "dbuser", config.User)
	assert.Equal(t, "dbpass", config.Passwd)
	assert.Equal(t, "dbname", config.DBName)
	assert.True(t, config.WriteData)
	assert.True(t, config.UseBlosc)
	assert.Equal(t, "zstd", config.BloscAlgorithm)
	assert.Equal(t, 9, config.CompressionLevel)
	assert.Equal(t, "no", config.BitShuffle)
}

func TestDecoderConfiguration_JSONSerialization(t *testing.T) {
	config := DecoderConfiguration{
		ExtTrigger:       1,
		TrgCode1:         10,
		TrgCode2:         20,
		ReadPMTs:         true,
		ReadSiPMs:        false,
		ReadTrigger:      true,
		SplitTrigger:     false,
		NoDB:             false,
		Discard:          false,
		Host:             "localhost",
		User:             "user",
		Passwd:           "pass",
		DBName:           "testdb",
		WriteData:        true,
		UseBlosc:         true,
		BloscAlgorithm:   "lz4",
		CompressionLevel: 5,
		BitShuffle:       "yes",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(config)
	require.NoError(t, err)

	// Verify JSON structure uses correct tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	// Check that JSON keys match the json tags
	assert.Equal(t, float64(1), jsonMap["ext_trigger"])
	assert.Equal(t, float64(10), jsonMap["trg_code_1"])
	assert.Equal(t, float64(20), jsonMap["trg_code_2"])
	assert.Equal(t, true, jsonMap["read_pmts"])
	assert.Equal(t, false, jsonMap["read_sipms"])
	assert.Equal(t, true, jsonMap["read_trigger"])
	assert.Equal(t, false, jsonMap["split_trigger"])
	assert.Equal(t, false, jsonMap["no_db"])
	assert.Equal(t, false, jsonMap["discard"])
	assert.Equal(t, "localhost", jsonMap["host"])
	assert.Equal(t, "user", jsonMap["user"])
	assert.Equal(t, "pass", jsonMap["password"]) // Note: json tag is "password", not "passwd"
	assert.Equal(t, "testdb", jsonMap["db_name"])
	assert.Equal(t, true, jsonMap["write_data"])
	assert.Equal(t, true, jsonMap["use_blosc"])
	assert.Equal(t, "lz4", jsonMap["blosc_algorithm"])
	assert.Equal(t, float64(5), jsonMap["compression_level"])
	assert.Equal(t, "yes", jsonMap["bit_shuffle"])
}

func TestDecoderConfiguration_JSONRoundtrip(t *testing.T) {
	original := DecoderConfiguration{
		ExtTrigger:       5,
		TrgCode1:         15,
		TrgCode2:         25,
		ReadPMTs:         true,
		ReadSiPMs:        true,
		ReadTrigger:      false,
		SplitTrigger:     true,
		NoDB:             true,
		Discard:          true,
		Host:             "dbserver",
		User:             "admin",
		Passwd:           "secret",
		DBName:           "production",
		WriteData:        false,
		UseBlosc:         false,
		BloscAlgorithm:   "snappy",
		CompressionLevel: 3,
		BitShuffle:       "no",
	}

	// Marshal
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var restored DecoderConfiguration
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original, restored)
}

func TestGetDecoderParameters_ContextPropagation(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	dbResult := database.Decoderparam{
		ExtTrigger:       1,
		TrgCode1:         1,
		TrgCode2:         1,
		ReadPmts:         sql.NullBool{Bool: true, Valid: true},
		ReadSipms:        sql.NullBool{Bool: true, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: true, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: true, Valid: true},
		NoDb:             sql.NullBool{Bool: true, Valid: true},
		Discard:          sql.NullBool{Bool: true, Valid: true},
		Host:             "host",
		User:             "user",
		Passwd:           "pass",
		DbName:           "db",
		WriteData:        sql.NullBool{Bool: true, Valid: true},
		UseBlosc:         sql.NullBool{Bool: true, Valid: true},
		BloscAlgorithm:   "lz4",
		CompressionLevel: 1,
		BitShuffle:       "yes",
	}

	// Verify that context.Background() is passed
	mockQuerier.On("GetDecoderParams", context.Background()).Return(dbResult, nil)

	config, err := getDecoderParameters(mockQuerier)
	require.NoError(t, err)

	mockQuerier.AssertExpectations(t)

	// Verify the config values match the input
	assert.Equal(t, 1, config.ExtTrigger)
	assert.Equal(t, 1, config.TrgCode1)
	assert.Equal(t, 1, config.TrgCode2)
	assert.True(t, config.ReadPMTs)
	assert.True(t, config.ReadSiPMs)
	assert.True(t, config.ReadTrigger)
	assert.True(t, config.SplitTrigger)
	assert.True(t, config.NoDB)
	assert.True(t, config.Discard)
	assert.Equal(t, "host", config.Host)
	assert.Equal(t, "user", config.User)
	assert.Equal(t, "pass", config.Passwd)
	assert.Equal(t, "db", config.DBName)
	assert.True(t, config.WriteData)
	assert.True(t, config.UseBlosc)
	assert.Equal(t, "lz4", config.BloscAlgorithm)
	assert.Equal(t, 1, config.CompressionLevel)
	assert.Equal(t, "yes", config.BitShuffle)
}

func TestGetDecoderParameters_ZeroValues(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	// Test with all zero/empty values
	dbResult := database.Decoderparam{
		ExtTrigger:       0,
		TrgCode1:         0,
		TrgCode2:         0,
		ReadPmts:         sql.NullBool{Bool: false, Valid: true},
		ReadSipms:        sql.NullBool{Bool: false, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: false, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: false, Valid: true},
		NoDb:             sql.NullBool{Bool: false, Valid: true},
		Discard:          sql.NullBool{Bool: false, Valid: true},
		Host:             "",
		User:             "",
		Passwd:           "",
		DbName:           "",
		WriteData:        sql.NullBool{Bool: false, Valid: true},
		UseBlosc:         sql.NullBool{Bool: false, Valid: true},
		BloscAlgorithm:   "",
		CompressionLevel: 0,
		BitShuffle:       "",
	}

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(dbResult, nil)

	config, err := getDecoderParameters(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, 0, config.ExtTrigger)
	assert.Equal(t, 0, config.TrgCode1)
	assert.Equal(t, 0, config.TrgCode2)
	assert.False(t, config.ReadPMTs)
	assert.False(t, config.ReadSiPMs)
	assert.False(t, config.ReadTrigger)
	assert.False(t, config.SplitTrigger)
	assert.False(t, config.NoDB)
	assert.False(t, config.Discard)
	assert.Equal(t, "", config.User)
	assert.Equal(t, "", config.Passwd)
	assert.Equal(t, "", config.DBName)
	assert.False(t, config.WriteData)
	assert.False(t, config.UseBlosc)
	assert.Equal(t, "", config.BloscAlgorithm)
	assert.Equal(t, "", config.BitShuffle)
	assert.Equal(t, "", config.Host)
	assert.Equal(t, 0, config.CompressionLevel)
}
