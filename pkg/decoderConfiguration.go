package duck

import (
	"context"

	"github.com/jmbenlloch/next_duck/pkg/database"
)

type DecoderConfiguration struct {
	ExtTrigger       int    `json:"ext_trigger" db:"ext_trigger"`
	TrgCode1         int    `json:"trg_code_1" db:"trg_code_1"`
	TrgCode2         int    `json:"trg_code_2" db:"trg_code_2"`
	ReadPMTs         bool   `json:"read_pmts" db:"read_pmts"`
	ReadSiPMs        bool   `json:"read_sipms" db:"read_sipms"`
	ReadTrigger      bool   `json:"read_trigger" db:"read_trigger"`
	SplitTrigger     bool   `json:"split_trigger" db:"split_trigger"`
	NoDB             bool   `json:"no_db" db:"no_db"`
	Discard          bool   `json:"discard" db:"discard"`
	Host             string `json:"host" db:"host"`
	User             string `json:"user" db:"user"`
	Passwd           string `json:"password" db:"passwd"`
	DBName           string `json:"db_name" db:"db_name"`
	WriteData        bool   `json:"write_data" db:"write_data"`
	UseBlosc         bool   `json:"use_blosc" db:"use_blosc"`
	BloscAlgorithm   string `json:"blosc_algorithm" db:"blosc_algorithm"`
	CompressionLevel int    `json:"compression_level" db:"compression_level"`
	BitShuffle       string `json:"bit_shuffle" db:"bit_shuffle"`
}

func getDecoderParameters(queries database.Querier) (DecoderConfiguration, error) {
	config, err := queries.GetDecoderParams(context.Background())
	if err != nil {
		return DecoderConfiguration{}, err
	}

	return DecoderConfiguration{
		ExtTrigger:       int(config.ExtTrigger),
		TrgCode1:         int(config.TrgCode1),
		TrgCode2:         int(config.TrgCode2),
		ReadPMTs:         config.ReadPmts.Bool,
		ReadSiPMs:        config.ReadSipms.Bool,
		ReadTrigger:      config.ReadTrigger.Bool,
		SplitTrigger:     config.SplitTrigger.Bool,
		NoDB:             config.NoDb.Bool,
		Discard:          config.Discard.Bool,
		Host:             config.Host,
		User:             config.User,
		Passwd:           config.Passwd,
		DBName:           config.DbName,
		WriteData:        config.WriteData.Bool,
		UseBlosc:         config.UseBlosc.Bool,
		BloscAlgorithm:   config.BloscAlgorithm,
		CompressionLevel: int(config.CompressionLevel),
		BitShuffle:       config.BitShuffle,
	}, nil
}
