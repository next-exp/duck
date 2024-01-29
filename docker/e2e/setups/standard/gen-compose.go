//go:build !next100
// +build !next100

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	setupName     = "standard"
	setupDir      = "docker/e2e/setups/standard"
	setupInitDir  = "docker/e2e/setups/standard/mysql/init"
	commonInitDir = "docker/e2e/mysql/init"
)

type cfg struct {
	L, D, G                  int
	DataPrefix               string
	LdcIP, DevIPStart        int
	EqPortBase               int
	GdcTCPPortBase           int
	LdcGRPCPortBase          int
	GdcGRPCPortBase          int
	LdcPromBase, GdcPromBase int
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var x int
		fmt.Sscanf(v, "%d", &x)
		return x
	}
	return def
}

func getenvStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	c := cfg{
		L:               getenvInt("L", 1),
		D:               getenvInt("D", 1),
		G:               getenvInt("G", 1),
		DataPrefix:      getenvStr("DATA_SUBNET_PREFIX", "172.30"),
		LdcIP:           getenvInt("LDC_DATA_IP_SUFFIX", 10),
		DevIPStart:      getenvInt("DEV_DATA_IP_START", 20),
		EqPortBase:      getenvInt("EQUIPMENT_PORT_BASE", 6000),
		GdcTCPPortBase:  getenvInt("GDC_TCP_PORT_BASE", 6100),
		LdcGRPCPortBase: getenvInt("LDC_GRPC_PORT_BASE", 50050),
		GdcGRPCPortBase: getenvInt("GDC_GRPC_PORT_BASE", 50060),
		LdcPromBase:     getenvInt("LDC_PROM_PORT_BASE", 12110),
		GdcPromBase:     getenvInt("GDC_PROM_PORT_BASE", 12120),
	}

	// Ensure setup directory exists
	os.MkdirAll(setupInitDir, 0755)

	// Copy common init files
	copyCommonInitFiles()

	genCompose(c)
	genSeedSQL(c)

	fmt.Printf("Generated %s setup in %s/\n", setupName, setupInitDir)
}

func copyCommonInitFiles() {
	files := []string{"01-schema.sql", "02-users.sql"}
	for _, f := range files {
		src := filepath.Join(commonInitDir, f)
		dst := filepath.Join(setupInitDir, f)
		copyFile(src, dst)
	}
}

func copyFile(src, dst string) {
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

func genCompose(c cfg) {
	var y string
	y += "networks:\n  daq_ctrl: {external: false}\n"
	for i := 1; i <= c.L; i++ {
		y += fmt.Sprintf("  data_ldc_%d:\n    driver: bridge\n    ipam:\n      config:\n        - subnet: %s.%d.0/24\n", i, c.DataPrefix, i)
	}
	y += "\nservices:\n"

	// Override mysql to use setup-specific init directory
	y += "  mysql:\n"
	y += "    volumes:\n"
	y += "      - mysql_data:/var/lib/mysql\n"
	y += "      - ./setups/standard/mysql/init:/docker-entrypoint-initdb.d\n"

	// GDCs
	for k := 1; k <= c.G; k++ {
		name := fmt.Sprintf("gdc-%d", k)
		y += fmt.Sprintf("  %s:\n    image: duck-e2e:latest\n    hostname: %s\n    working_dir: /app\n    command: [\"/app/bin/gdcRPC\", \"-config\", \"/app/docker/e2e/configuration.yml\", \"-name\", \"%s\"]\n    depends_on: { mysql: { condition: service_healthy }, centrifugo: { condition: service_healthy } }\n    volumes:\n      - ../data/%s:/data\n    networks: [daq_ctrl]\n", name, name, name, name)
	}
	// LDCs + devices
	for i := 1; i <= c.L; i++ {
		ldc := fmt.Sprintf("ldc-%d", i)
		y += fmt.Sprintf("  %s:\n    image: duck-e2e:latest\n    hostname: %s\n    working_dir: /app\n    command: [\"/app/bin/ldcRPC\", \"-config\", \"/app/docker/e2e/configuration.yml\"]\n    depends_on: { mysql: { condition: service_healthy }, centrifugo: { condition: service_healthy } }\n    networks:\n      daq_ctrl: {}\n      data_ldc_%d:\n        ipv4_address: %s.%d.%d\n", ldc, ldc, i, c.DataPrefix, i, c.LdcIP)
		for j := 1; j <= c.D; j++ {
			dev := fmt.Sprintf("sim-%d-%d", i, j)
			eqID := (i-1)*c.D + j
			devIP := c.DevIPStart + j
			y += fmt.Sprintf("  %s:\n    image: duck-e2e:latest\n    hostname: %s\n    working_dir: /app\n    command: [\"/app/bin/deviceSimulator\", \"-config\", \"/app/docker/e2e/configuration.yml\", \"-id\", \"%d\", \"-mode\", \"generate\"]\n    depends_on: { mysql: { condition: service_healthy }, centrifugo: { condition: service_healthy } }\n    environment:\n      - DB_HOST=mysql\n      - DB_USER=duck\n      - DB_PASS=dummy_password\n      - DB_PORT=3306\n      - DB_NAME=duck\n    networks:\n      daq_ctrl: {}\n      data_ldc_%d:\n        ipv4_address: %s.%d.%d\n", dev, dev, eqID, i, c.DataPrefix, i, devIP)
		}
	}
	os.WriteFile(filepath.Join(setupDir, "docker-compose.gen.yml"), []byte(y), 0644)
}

func genSeedSQL(c cfg) {
	s := "USE duck;\n\n"
	s += "DELETE FROM testDeviceParams;\nDELETE FROM equipments;\nDELETE FROM ldcs;\nDELETE FROM gdcs;\nDELETE FROM runs;\nDELETE FROM duckParams;\nDELETE FROM decoderParams;\n\n"
	// base params
	s += "INSERT INTO runs (id, start, stop) VALUES (1, NULL, NULL);\n"
	s += "INSERT INTO duckParams (filesize, experiment, packetSize, nPacketsInBuffer, buffertimeout, equipmentDataCh, receiveCh, dataCh, metricsCh, tcpConnectionsCh, decoderCh, decoderWorkers, writerCh) VALUES (100000000, 'DEV', 500, 4096, 30, 1024, 1024, 8192, 1024, 64, 1024, 2, 1024);\n"
	s += "INSERT INTO decoderParams (ext_trigger, trg_code_1, trg_code_2, read_pmts, read_sipms, read_trigger, split_trigger, no_db, discard, host, user, passwd, db_name, write_data, use_blosc, blosc_algorithm, compression_level, bit_shuffle) VALUES (0,0,0,false,false,false,false,true,false,'localhost','duck','dummy_password','duck',false,false,'lz4',1,'none');\n\n"
	// GDCs
	for k := 1; k <= c.G; k++ {
		name := fmt.Sprintf("gdc-%d", k)
		id := k
		port := c.GdcTCPPortBase + k - 1
		grpc := c.GdcGRPCPortBase + k - 1
		prom := c.GdcPromBase + k - 1
		s += fmt.Sprintf("INSERT INTO gdcs (id, name, hostname, ip, port, grpc_port, prometheus_port, datapath, enabled, writeOutput, decode) VALUES (%d, '%s', '%s', '%s', %d, %d, %d, '/data', true, true, false);\n", id, name, name, name, port, grpc, prom)
	}
	// LDCs and equipments
	for i := 1; i <= c.L; i++ {
		name := fmt.Sprintf("ldc-%d", i)
		id := i
		grpc := c.LdcGRPCPortBase + i - 1
		prom := c.LdcPromBase + i - 1
		s += fmt.Sprintf("INSERT INTO ldcs (id, name, hostname, ip, grpc_port, prometheus_port, enabled) VALUES (%d, '%s', '%s', '%s', %d, %d, true);\n", id, name, name, name, grpc, prom)
		for j := 1; j <= c.D; j++ {
			eqID := (i-1)*c.D + j
			devIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, i, c.DevIPStart+j)
			ldcDataIP := fmt.Sprintf("%s.%d.%d", c.DataPrefix, i, c.LdcIP)
			port := c.EqPortBase + j - 1
			s += fmt.Sprintf("INSERT INTO equipments (id, type, device_ip, host_ip, host_port, enabled, ldcID) VALUES (%d, 22, '%s', '%s', %d, true, %d);\n", eqID, devIP, ldcDataIP, port, id)
			// Insert simulator parameters with default values
			s += fmt.Sprintf("INSERT INTO simulatorParams (equipmentID, filePath, rateHz, loopMode, maxEvents, packetSize) VALUES (%d, '/app/data/test-data.rd', 0.1, true, 0, 500);\n", eqID)
		}
	}
	os.WriteFile(filepath.Join(setupInitDir, "03-seed.gen.sql"), []byte(s), 0644)
}
