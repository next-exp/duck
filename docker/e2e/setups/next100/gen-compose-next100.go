//go:build next100
// +build next100

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// This generator creates a docker-compose setup that replicates the real NEXT-100 DAQ system
// It uses device simulators with real data files instead of test devices

const (
	next100SetupName     = "next100"
	next100SetupDir      = "docker/e2e/setups/next100"
	next100SetupInitDir  = "docker/e2e/setups/next100/mysql/init"
	next100CommonInitDir = "docker/e2e/mysql/init"
	next100DecoderDir    = "docker/e2e/mysql/decoder"
)

type next100Config struct {
	DataFile        string
	FragmentSize    int
	DataPrefix      string
	CtrlPrefix      string
	LdcDataIPSuffix int
	DevDataIPStart  int
	EqPortBase      int
	GdcTCPPortBase  int
	LdcGRPCPortBase int
	GdcGRPCPortBase int
	SimGRPCPortBase int
	LdcPromBase     int
	GdcPromBase     int
	SimPromBase     int
}

func getenvStrN100(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvIntN100(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var x int
		fmt.Sscanf(v, "%d", &x)
		return x
	}
	return def
}

func main() {
	c := next100Config{
		DataFile:        getenvStrN100("NEXT100_DATA_FILE", "/app/next100/run_14816.ldc2next.next-100.042.rd"),
		FragmentSize:    getenvIntN100("NEXT100_FRAGMENT_SIZE", 7992),
		DataPrefix:      getenvStrN100("DATA_SUBNET_PREFIX", "172.30"),
		CtrlPrefix:      getenvStrN100("CTRL_SUBNET_PREFIX", "172.31"),
		LdcDataIPSuffix: getenvIntN100("LDC_DATA_IP_SUFFIX", 10),
		DevDataIPStart:  getenvIntN100("DEV_DATA_IP_START", 20),
		EqPortBase:      getenvIntN100("EQUIPMENT_PORT_BASE", 6000),
		GdcTCPPortBase:  getenvIntN100("GDC_TCP_PORT_BASE", 6100),
		LdcGRPCPortBase: getenvIntN100("LDC_GRPC_PORT_BASE", 50050),
		GdcGRPCPortBase: getenvIntN100("GDC_GRPC_PORT_BASE", 50060),
		SimGRPCPortBase: getenvIntN100("SIM_GRPC_PORT_BASE", 50070),
		LdcPromBase:     getenvIntN100("LDC_PROM_PORT_BASE", 12110),
		GdcPromBase:     getenvIntN100("GDC_PROM_PORT_BASE", 12120),
		SimPromBase:     getenvIntN100("SIM_PROM_PORT_BASE", 12130),
	}

	// Ensure setup directory exists
	os.MkdirAll(next100SetupInitDir, 0755)

	// Copy common init files
	copyCommonInitFilesNext100()

	genComposeNext100(c)
	genSeedSQLNext100(c)

	fmt.Printf("Generated %s setup in %s/\n", next100SetupName, next100SetupInitDir)
}

func copyCommonInitFilesNext100() {
	// Clean up old decoder files from previous runs
	oldFiles := []string{
		"04-00-create-next100db.sql",
		"04-ChannelMapping.sql",
		"04-HuffmanCodesPmt.sql",
		"04-HuffmanCodesSipm.sql",
	}
	for _, f := range oldFiles {
		os.Remove(filepath.Join(next100SetupInitDir, f))
	}

	files := []string{"01-schema.sql", "02-users.sql"}
	for _, f := range files {
		src := filepath.Join(next100CommonInitDir, f)
		dst := filepath.Join(next100SetupInitDir, f)
		copyFileNext100(src, dst)
	}

	// Copy decoder SQL files for NEXT100DB (ordered by prefix)
	decoderFiles := []struct {
		src string
		dst string
	}{
		{"00-create-next100db.sql", "04-create-next100db.sql"},
		{"ChannelMapping.sql", "05-ChannelMapping.sql"},
		{"HuffmanCodesPmt.sql", "06-HuffmanCodesPmt.sql"},
		{"HuffmanCodesSipm.sql", "07-HuffmanCodesSipm.sql"},
	}
	for _, f := range decoderFiles {
		src := filepath.Join(next100DecoderDir, f.src)
		dst := filepath.Join(next100SetupInitDir, f.dst)
		copyFileNext100(src, dst)
	}
}

func copyFileNext100(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open %s: %v\n", src, err)
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create %s: %v\n", dst, err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not copy %s to %s: %v\n", src, dst, err)
	}
}

func genComposeNext100(c next100Config) {
	// NEXT-100 real configuration:
	// - 7 LDCs (ldc1next - ldc7next)
	// - 7 GDCs (gdc1next - gdc7next)
	// - 4 equipments per LDC (28 total) with real equipment IDs from data file

	// Real equipment IDs from run_14816.ldc2next.next-100.042.rd:
	// LDC 1: 16, 17, 18, 19
	// LDC 2: 101, 102, 103 (only 3 equipments!)
	// LDC 3: 25, 26, 27, 28
	// LDC 4: 208, 209, 210, 211
	// LDC 5: 216, 217, 218, 219
	// LDC 6: 220, 221, 222, 223
	// LDC 7: 224, 225, 226, 227

	var y string

	// Networks
	y += "networks:\n"
	y += "  daq_ctrl:\n    external: false\n"
	y += "  daq_ctrl_next100:\n    driver: bridge\n    ipam:\n      config:\n        - subnet: " + c.CtrlPrefix + ".0.0/16\n"

	for i := 1; i <= 7; i++ {
		y += fmt.Sprintf("  data_ldc_%d:\n    driver: bridge\n    ipam:\n      config:\n        - subnet: %s.%d.0/24\n", i, c.DataPrefix, i)
	}

	y += "\nservices:\n"

	// Override mysql to use setup-specific init directory
	y += "  mysql:\n"
	y += "    volumes:\n"
	y += "      - mysql_data:/var/lib/mysql\n"
	y += "      - ./setups/next100/mysql/init:/docker-entrypoint-initdb.d\n"

	// API service override - connect to both networks
	y += "  api:\n"
	y += "    networks:\n"
	y += "      - daq_ctrl\n"
	y += "      - daq_ctrl_next100\n"

	// GDCs (7 total: gdc1next - gdc7next)
	for k := 1; k <= 7; k++ {
		name := fmt.Sprintf("gdc%dnext", k)
		ctrlIP := fmt.Sprintf("%s.11.%d", c.CtrlPrefix, k)

		y += fmt.Sprintf("  %s:\n", name)
		y += "    image: duck-e2e:latest\n"
		y += fmt.Sprintf("    hostname: %s\n", name)
		y += "    working_dir: /app\n"
		y += fmt.Sprintf("    command: [\"/app/bin/gdcRPC\", \"-config\", \"/app/docker/e2e/configuration.yml\", \"-name\", \"%s\"]\n", name)
		y += "    depends_on:\n      mysql: { condition: service_healthy }\n      centrifugo: { condition: service_healthy }\n"
		y += fmt.Sprintf("    volumes:\n      - ../data/%s:/data\n", name)
		y += "    networks:\n"
		y += "      daq_ctrl: {}\n"
		y += fmt.Sprintf("      daq_ctrl_next100:\n        ipv4_address: %s\n", ctrlIP)
	}

	// LDCs + Device Simulators with real equipment IDs
	// Map of LDC ID -> equipment IDs from the real data file
	ldcEquipments := map[int][]int{
		1: {16, 17, 18, 19},
		2: {101, 102, 103}, // LDC 2 only has 3 equipments
		3: {25, 26, 27, 28},
		4: {208, 209, 210, 211},
		5: {216, 217, 218, 219},
		6: {220, 221, 222, 223},
		7: {224, 225, 226, 227},
	}

	for ldcID := 1; ldcID <= 7; ldcID++ {
		ldcName := fmt.Sprintf("ldc%dnext", ldcID)
		ldcCtrlIP := fmt.Sprintf("%s.10.%d", c.CtrlPrefix, ldcID)
		ldcDataIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, ldcID, c.LdcDataIPSuffix)

		// LDC
		y += fmt.Sprintf("  %s:\n", ldcName)
		y += "    image: duck-e2e:latest\n"
		y += fmt.Sprintf("    hostname: %s\n", ldcName)
		y += "    working_dir: /app\n"
		y += "    command: [\"/app/bin/ldcRPC\", \"-config\", \"/app/docker/e2e/configuration.yml\"]\n"
		y += "    depends_on:\n      mysql: { condition: service_healthy }\n      centrifugo: { condition: service_healthy }\n"
		y += "    networks:\n"
		y += "      daq_ctrl: {}\n"
		y += fmt.Sprintf("      daq_ctrl_next100:\n        ipv4_address: %s\n", ldcCtrlIP)
		y += fmt.Sprintf("      data_ldc_%d:\n        ipv4_address: %s\n", ldcID, ldcDataIP)

		// Device Simulators with real equipment IDs
		equipments := ldcEquipments[ldcID]
		for idx, eqID := range equipments {
			simName := fmt.Sprintf("sim-%d-%d", ldcID, idx+1)
			devDataIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, ldcID, c.DevDataIPStart+idx+1)

			y += fmt.Sprintf("  %s:\n", simName)
			y += "    image: duck-e2e:latest\n"
			y += fmt.Sprintf("    hostname: %s\n", simName)
			y += "    working_dir: /app\n"
			y += fmt.Sprintf("    command: [\"/app/bin/deviceSimulator\", \"-config\", \"/app/docker/e2e/configuration.yml\", \"-id\", \"%d\"]\n", eqID)
			y += "    depends_on:\n      mysql: { condition: service_healthy }\n      centrifugo: { condition: service_healthy }\n"
			y += "    environment:\n      DB_HOST: mysql\n      DB_USER: duck\n      DB_PASS: dummy_password\n      DB_PORT: 3306\n      DB_NAME: duck\n"
			y += "    networks:\n"
			y += "      daq_ctrl: {}\n"
			y += fmt.Sprintf("      data_ldc_%d:\n        ipv4_address: %s\n", ldcID, devDataIP)
		}
	}

	os.WriteFile(filepath.Join(next100SetupDir, "docker-compose.gen-next100.yml"), []byte(y), 0644)
	fmt.Println("Generated docker-compose.gen-next100.yml")
}

func genSeedSQLNext100(c next100Config) {
	s := "USE duck;\n\n"
	// Delete in correct order to respect foreign key constraints
	s += "DELETE FROM simulatorParams;\n"
	s += "DELETE FROM testDeviceParams;\n"
	s += "DELETE FROM equipments;\n"
	s += "DELETE FROM ldcs;\n"
	s += "DELETE FROM gdcs;\n"
	s += "DELETE FROM runs;\n"
	s += "DELETE FROM duckParams;\n"
	s += "DELETE FROM decoderParams;\n\n"

	// Base configuration
	s += "INSERT INTO runs (id, start, stop) VALUES (14815, NULL, NULL);\n\n"

	// Duck params for NEXT-100
	s += "INSERT INTO duckParams (filesize, experiment, packetSize, nPacketsInBuffer, buffertimeout, equipmentDataCh, receiveCh, dataCh, metricsCh, tcpConnectionsCh, decoderCh, decoderWorkers, writerCh) VALUES (500000000, 'next-100', 9500, 50000, 30, 10000, 10000, 1000, 1000, 1000, 1000, 15, 1000);\n\n"

	// Decoder params for NEXT-100 - use the e2e MySQL server with NEXT100DB database
	s += "INSERT INTO decoderParams (ext_trigger, trg_code_1, trg_code_2, read_pmts, read_sipms, read_trigger, split_trigger, no_db, discard, host, user, passwd, db_name, write_data, use_blosc, blosc_algorithm, compression_level, bit_shuffle) VALUES (15,1,9,1,1,1,1,0,0,'mysql','duck','dummy_password','NEXT100DB',0,1,'lz4',4,'bit-shuffle');\n\n"

	// GDCs (7 total: gdc1next - gdc7next)
	for k := 1; k <= 7; k++ {
		name := fmt.Sprintf("gdc%dnext", k)
		ctrlIP := fmt.Sprintf("%s.11.%d", c.CtrlPrefix, k)
		tcpPort := c.GdcTCPPortBase + k - 1
		grpcPort := c.GdcGRPCPortBase + k - 1
		promPort := c.GdcPromBase + k - 1

		s += fmt.Sprintf("INSERT INTO gdcs (id, name, hostname, ip, port, grpc_port, prometheus_port, datapath, enabled, writeOutput, decode) VALUES (%d, '%s', '%s', '%s', %d, %d, %d, '/data', true, true, true);\n", k, name, name, ctrlIP, tcpPort, grpcPort, promPort)
	}
	s += "\n"

	// LDCs and Equipments with real equipment IDs
	// Map of LDC ID -> equipment IDs from the real data file
	ldcEquipments := map[int][]int{
		1: {16, 17, 18, 19},
		2: {101, 102, 103}, // LDC 2 only has 3 equipments
		3: {25, 26, 27, 28},
		4: {208, 209, 210, 211},
		5: {216, 217, 218, 219},
		6: {220, 221, 222, 223},
		7: {224, 225, 226, 227},
	}

	for ldcID := 1; ldcID <= 7; ldcID++ {
		ldcName := fmt.Sprintf("ldc%dnext", ldcID)
		ldcCtrlIP := fmt.Sprintf("%s.10.%d", c.CtrlPrefix, ldcID)
		ldcDataIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, ldcID, c.LdcDataIPSuffix)
		grpcPort := c.LdcGRPCPortBase + ldcID - 1
		promPort := c.LdcPromBase + ldcID - 1

		// Insert LDC
		s += fmt.Sprintf("INSERT INTO ldcs (id, name, hostname, ip, grpc_port, prometheus_port, enabled) VALUES (%d, '%s', '%s', '%s', %d, %d, true);\n", ldcID, ldcName, ldcName, ldcCtrlIP, grpcPort, promPort)

		// Insert equipments with real IDs
		equipments := ldcEquipments[ldcID]
		for idx, eqID := range equipments {
			devDataIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, ldcID, c.DevDataIPStart+idx+1)
			port := c.EqPortBase + idx

			s += fmt.Sprintf("INSERT INTO equipments (id, type, device_ip, host_ip, host_port, enabled, ldcID) VALUES (%d, 22, '%s', '%s', %d, true, %d);\n", eqID, devDataIP, ldcDataIP, port, ldcID)
		}
	}
	s += "\n"

	// Insert simulator parameters for all devices with real equipment IDs
	for ldcID := 1; ldcID <= 7; ldcID++ {
		equipments := ldcEquipments[ldcID]
		for _, eqID := range equipments {
			s += fmt.Sprintf("INSERT INTO simulatorParams (equipmentID, filePath, rateHz, loopMode, maxEvents, packetSize) VALUES (%d, '%s', 0.1, true, 0, %d);\n", eqID, c.DataFile, c.FragmentSize)
		}
	}

	// Write to setup-specific directory
	os.WriteFile(filepath.Join(next100SetupInitDir, "03-seed.gen.sql"), []byte(s), 0644)
	fmt.Println("Generated 03-seed.gen.sql")
}
