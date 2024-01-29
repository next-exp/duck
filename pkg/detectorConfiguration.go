package duck

import (
	"context"
	"errors"
	"os"

	"github.com/jmbenlloch/next_duck/pkg/database"
)

type Configuration struct {
	RunNumber        int                `yaml:"run_number"`
	LDCs             []LDCConfiguration `yaml:"ldcs"`
	GDCs             []GDCConfiguration `yaml:"gdcs"`
	Duck             DuckConfiguration
	Decoder          DecoderConfiguration
	TestDeviceParams []TestDeviceConfiguration
}

type DuckConfiguration struct {
	Filesize         int    `db:"filesize"`
	Experiment       string `db:"experiment"`
	PacketSize       int    `db:"packetSize"`
	NPacketsInBuffer int    `db:"nPacketsInBuffer"`
	BufferTimeout    int    `db:"bufferTimeout"`
	EquipmentCh      int    `db:"equipmentDataCh"`
	ReceiveCh        int    `db:"receiveCh"`
	DataCh           int    `db:"dataCh"`
	MetricsCh        int    `db:"metricsCh"`
	TCPConnectionsCh int    `db:"tcpConnectionsCh"`
	WriterCh         int    `db:"writerCh"`
	DecoderCh        int    `db:"decoderCh"`
	DecoderWorkers   int    `db:"decoderWorkers"`
}

type LDCConfiguration struct {
	ID             int    `db:"id"`
	Name           string `db:"name"`
	Host           string `db:"hostname"`
	IP             string `db:"ip"`
	GRPCPort       int    `db:"grpc_port"`
	PrometheusPort int    `db:"prometheus_port"`
	Enabled        bool   `db:"enabled"`
	Equipments     []Equipment
}

type GDCConfiguration struct {
	ID             int    `db:"id"`
	IP             string `db:"ip"`
	Port           int    `db:"port"`
	GRPCPort       int    `db:"grpc_port"`
	PrometheusPort int    `db:"prometheus_port"`
	Name           string `db:"name"`
	Host           string `db:"hostname"`
	Path           string `db:"datapath"`
	Enabled        bool   `db:"enabled"`
	WriteOutput    bool   `db:"writeOutput"`
	Decode         bool   `db:"decode"`
}

type Equipment struct {
	ID       int    `db:"id"`
	Type     int    `db:"type"`
	DeviceIP string `db:"device_ip"`
	HostIP   string `db:"host_ip"`
	HostPort int    `db:"host_port"`
	LDC_ID   int    `db:"ldcID"`
	Enabled  bool   `db:"enabled"`
}

type TestDeviceConfiguration struct {
	EquipmentID        int     `db:"equipment_id"`
	EventRateMs        int     `db:"event_rate_ms"`
	PacketsPerEvent    int     `db:"packets_per_event"`
	PacketSize         int     `db:"packet_size"`
	ErrorInjectionRate float64 `db:"error_injection_rate"`
	MaxEvents          int     `db:"max_events"`
}

func getGDCs(queries database.Querier) ([]GDCConfiguration, error) {
	gdcsList, err := queries.ListGDCs(context.Background())
	var gdcs []GDCConfiguration

	if err != nil {
		return []GDCConfiguration{}, err
	}

	for _, gdc := range gdcsList {
		result := GDCConfiguration{
			ID:             int(gdc.ID),
			IP:             gdc.Ip.String,
			Port:           int(gdc.Port.Int32),
			GRPCPort:       int(gdc.GrpcPort.Int32),
			PrometheusPort: int(gdc.PrometheusPort.Int32),
			Name:           gdc.Name.String,
			Host:           gdc.Hostname.String,
			Path:           gdc.Datapath.String,
			Enabled:        gdc.Enabled.Bool,
			WriteOutput:    gdc.Writeoutput.Bool,
			Decode:         gdc.Decode.Bool,
		}
		gdcs = append(gdcs, result)
	}

	return gdcs, nil
}

func getLDCs(queries database.Querier) ([]LDCConfiguration, error) {
	ldcsList, err := queries.ListLDCs(context.Background())
	var ldcs []LDCConfiguration

	if err != nil {
		return []LDCConfiguration{}, err
	}

	for _, ldc := range ldcsList {
		result := LDCConfiguration{
			ID:             int(ldc.ID),
			Name:           ldc.Name.String,
			Host:           ldc.Hostname.String,
			IP:             ldc.Ip.String,
			GRPCPort:       int(ldc.GrpcPort.Int32),
			PrometheusPort: int(ldc.PrometheusPort.Int32),
			Enabled:        ldc.Enabled.Bool,
			Equipments:     []Equipment{},
		}
		ldcs = append(ldcs, result)
	}

	return ldcs, nil
}

func getDuckParameters(queries database.Querier) (DuckConfiguration, error) {
	config, err := queries.GetDuckParams(context.Background())
	if err != nil {
		return DuckConfiguration{}, err
	}

	return DuckConfiguration{
		Filesize:         int(config.Filesize),
		Experiment:       config.Experiment,
		PacketSize:       int(config.Packetsize),
		NPacketsInBuffer: int(config.Npacketsinbuffer),
		BufferTimeout:    int(config.Buffertimeout),
		EquipmentCh:      int(config.Equipmentdatach),
		ReceiveCh:        int(config.Receivech),
		DataCh:           int(config.Datach),
		MetricsCh:        int(config.Metricsch),
		TCPConnectionsCh: int(config.Tcpconnectionsch),
		WriterCh:         int(config.Writerch),
		DecoderCh:        int(config.Decoderch),
		DecoderWorkers:   int(config.Decoderworkers),
	}, nil
}

func getRun(queries database.Querier) (int, error) {
	run, err := queries.GetLatestRun(context.Background())
	return int(run), err
}

func getEquipments(queries database.Querier) ([]Equipment, error) {
	equipmentsList, err := queries.ListEquipments(context.Background())
	var equipments []Equipment

	if err != nil {
		return []Equipment{}, err
	}

	for _, equipment := range equipmentsList {
		result := Equipment{
			ID:       int(equipment.ID),
			Type:     int(equipment.Type.Int32),
			DeviceIP: equipment.DeviceIp.String,
			HostIP:   equipment.HostIp.String,
			HostPort: int(equipment.HostPort.Int32),
			LDC_ID:   int(equipment.Ldcid.Int32),
			Enabled:  equipment.Enabled.Bool,
		}
		equipments = append(equipments, result)
	}

	return equipments, nil
}

func assignEquipmentToLDC(ldcs []LDCConfiguration, equipments []Equipment) {
	for i := 0; i < len(equipments); i++ {
		ldcID := equipments[i].LDC_ID
		for j := 0; j < len(ldcs); j++ {
			if ldcID == ldcs[j].ID {
				ldcs[j].Equipments = append(ldcs[j].Equipments, equipments[i])
			}
		}
	}
}

func getTestDeviceParams(queries database.Querier) ([]TestDeviceConfiguration, error) {
	testDevicesList, err := queries.GetTestDeviceParams(context.Background())
	var testDevices []TestDeviceConfiguration

	if err != nil {
		return []TestDeviceConfiguration{}, err
	}

	for _, testDevice := range testDevicesList {
		result := TestDeviceConfiguration{
			EquipmentID:        int(testDevice.EquipmentID),
			EventRateMs:        int(testDevice.EventRateMs),
			PacketsPerEvent:    int(testDevice.PacketsPerEvent),
			PacketSize:         int(testDevice.PacketSize),
			ErrorInjectionRate: testDevice.ErrorInjectionRate,
			MaxEvents:          int(testDevice.MaxEvents),
		}
		testDevices = append(testDevices, result)
	}

	return testDevices, nil
}

func ReadConfigurationFromDB(queries database.Querier) (Configuration, error) {
	gdcs, err := getGDCs(queries)
	if err != nil {
		return Configuration{}, err
	}

	ldcs, err := getLDCs(queries)
	if err != nil {
		return Configuration{}, err
	}

	equipments, err := getEquipments(queries)
	if err != nil {
		return Configuration{}, err
	}

	run, err := getRun(queries)
	if err != nil {
		return Configuration{}, err
	}
	assignEquipmentToLDC(ldcs, equipments)

	duckConfig, err := getDuckParameters(queries)
	if err != nil {
		return Configuration{}, err
	}

	decoderConfig, err := getDecoderParameters(queries)
	if err != nil {
		return Configuration{}, err
	}

	testDeviceParams, err := getTestDeviceParams(queries)
	if err != nil {
		return Configuration{}, err
	}

	configuration := Configuration{
		RunNumber:        run,
		GDCs:             gdcs,
		LDCs:             ldcs,
		Duck:             duckConfig,
		Decoder:          decoderConfig,
		TestDeviceParams: testDeviceParams,
	}
	return configuration, nil
}

func GetGDCConfiguration(gdcs []GDCConfiguration, name string) (*GDCConfiguration, error) {
	Hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	var gdcConfiguration *GDCConfiguration = nil
	for idx := 0; idx < len(gdcs); idx++ {
		// Match by name first, then optionally verify hostname
		// For test environments, allow matching by name alone
		if gdcs[idx].Name == name {
			if gdcs[idx].Host == "" || gdcs[idx].Host == Hostname || gdcs[idx].Host == "127.0.0.1" {
				gdcConfiguration = &gdcs[idx]
				break
			}
		}
	}
	if gdcConfiguration == nil {
		err = errors.New("GDC not found in configuration")
	}
	return gdcConfiguration, err
}

func GetLDCConfiguration(ldcs []LDCConfiguration, name string) (*LDCConfiguration, error) {
	Hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	var ldcConfiguration *LDCConfiguration = nil
	for idx := 0; idx < len(ldcs); idx++ {
		// For empty name, return first matching LDC by hostname
		if name == "" {
			if ldcs[idx].Host == Hostname || ldcs[idx].Host == "127.0.0.1" || ldcs[idx].Host == "" {
				ldcConfiguration = &ldcs[idx]
				break
			}
		} else if ldcs[idx].Name == name {
			// Match by name first, then optionally verify hostname
			// For test environments, allow matching by name alone
			if ldcs[idx].Host == "" || ldcs[idx].Host == Hostname || ldcs[idx].Host == "127.0.0.1" {
				ldcConfiguration = &ldcs[idx]
				break
			}
		}
	}
	if ldcConfiguration == nil {
		err = errors.New("LDC not found in configuration")
	}
	return ldcConfiguration, err
}

func GetEquipmentConfiguration(ldcs []LDCConfiguration, equipmentID int) (*Equipment, error) {
	var equipmentConfiguration *Equipment = nil
	var err error = nil
	for idx := 0; idx < len(ldcs); idx++ {
		equipments := ldcs[idx].Equipments
		for idxEq := 0; idxEq < len(equipments); idxEq++ {
			if equipments[idxEq].ID == equipmentID {
				equipmentConfiguration = &equipments[idxEq]
				break
			}
		}
	}
	if equipmentConfiguration == nil {
		err = errors.New("equipment not found in configuration")
	}
	return equipmentConfiguration, err
}

func EnabledGDCs(gdcs []GDCConfiguration) []GDCConfiguration {
	enabledGDCs := make([]GDCConfiguration, 0)
	for _, gdc := range gdcs {
		if gdc.Enabled {
			enabledGDCs = append(enabledGDCs, gdc)
		}
	}
	return enabledGDCs
}

func EnabledLDCs(ldcs []LDCConfiguration) []LDCConfiguration {
	enabledLDCs := make([]LDCConfiguration, 0)
	for _, ldc := range ldcs {
		if ldc.Enabled {
			enabledLDCs = append(enabledLDCs, ldc)
		}
	}
	return enabledLDCs
}
