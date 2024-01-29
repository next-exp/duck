-- Integration Test Seed Data
-- Ensure testDeviceParams table exists for LDC config reads
-- Note: Table is created by schema.sql, this is kept for reference
CREATE TABLE IF NOT EXISTS testDeviceParams (
    equipment_id INT NOT NULL,
    event_rate_ms INT NOT NULL,
    packets_per_event INT NOT NULL,
    packet_size INT NOT NULL,
    error_injection_rate FLOAT NOT NULL,
    max_events INT NOT NULL
);

-- This file is loaded after the main schema (database/schema.sql)
-- It provides minimal test data for integration tests

-- GDCs (Global Data Collectors)
-- Hostname set to 127.0.0.1 to work with GetGDCConfiguration matching logic
INSERT INTO gdcs (id, name, hostname, ip, port, grpc_port, prometheus_port, datapath, enabled, writeOutput, decode)
VALUES
(1, 'test_gdc1', '127.0.0.1', '127.0.0.1', 6005, 50051, 12112, '/tmp/test_output', true, true, false),
(2, 'test_gdc2', '127.0.0.1', '127.0.0.1', 6006, 50052, 12113, '/tmp/test_output', true, true, false);

-- LDCs (Local Data Collectors)
-- Hostname set to 127.0.0.1 to work with GetLDCConfiguration matching logic
INSERT INTO ldcs (id, name, hostname, ip, grpc_port, prometheus_port, enabled)
VALUES
(1, 'test_ldc1', '127.0.0.1', '127.0.0.1', 50053, 12114, true),
(2, 'test_ldc2', '127.0.0.1', '127.0.0.1', 50054, 12115, true);

-- Equipments
INSERT INTO equipments (id, type, device_ip, host_ip, host_port, ldcID, enabled)
VALUES
(1, 22, '127.0.0.1', '127.0.0.1', 16006, 1, true),
(2, 22, '127.0.0.1', '127.0.0.1', 16007, 1, true),
(3, 22, '127.0.0.1', '127.0.0.1', 16008, 2, true),
(4, 22, '127.0.0.1', '127.0.0.1', 16009, 2, true);

-- Runs
INSERT INTO runs (id, start, stop) VALUES (1, null, null);

-- Duck parameters
INSERT INTO duckParams (filesize, experiment, packetSize, nPacketsInBuffer, buffertimeout, equipmentDataCh, receiveCh, dataCh, metricsCh, tcpConnectionsCh, writerCh, decoderCh, decoderWorkers)
VALUES (1000000000, 'integration_test', 1400, 1000, 30, 100, 100, 100, 100, 10, 100, 100, 4);

-- Decoder configurations
INSERT INTO decoderParams (ext_trigger, trg_code_1, trg_code_2, read_pmts, read_sipms, read_trigger, split_trigger, no_db, discard, host, user, passwd, db_name, write_data, use_blosc, blosc_algorithm, compression_level, bit_shuffle)
VALUES (0, 1, 2, true, true, true, false, false, false, 'localhost', 'test', 'test', 'test_db', true, true, 'lz4', 5, 'true');

-- Test Device Parameters (for equipment simulation/testing)
INSERT INTO testDeviceParams (equipment_id, event_rate_ms, packets_per_event, packet_size, error_injection_rate, max_events)
VALUES
(1, 10, 100, 1400, 0.0, 0),
(2, 15, 150, 1400, 0.01, 1000),
(3, 20, 200, 1500, 0.0, 0);
