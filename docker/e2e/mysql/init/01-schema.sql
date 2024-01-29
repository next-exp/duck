CREATE DATABASE IF NOT EXISTS duck;
CREATE DATABASE IF NOT EXISTS NEXT100DB;
USE duck;

CREATE TABLE IF NOT EXISTS runs (
    id INT PRIMARY KEY AUTO_INCREMENT NOT NULL,
    start TIMESTAMP NULL,
    stop TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS events (
    run INT NOT NULL,
    gdc_id INT NULL,
    ldc_id INT NULL,
    events BIGINT UNSIGNED
);

CREATE TABLE IF NOT EXISTS data (
    run INT NOT NULL,
    gdc_id INT NULL,
    ldc_id INT NULL,
    bytes BIGINT UNSIGNED
);

CREATE TABLE IF NOT EXISTS errors (
    run INT NOT NULL,
    gdc_id INT NULL,
    ldc_id INT NULL,
    errors BIGINT UNSIGNED
);

CREATE TABLE IF NOT EXISTS ldcs (
    id INT PRIMARY KEY AUTO_INCREMENT NOT NULL,
    name VARCHAR(100),
    hostname VARCHAR(100),
    ip VARCHAR(100),
    grpc_port INT,
    prometheus_port INT,
    enabled BOOLEAN
);

CREATE TABLE IF NOT EXISTS gdcs (
    id INT PRIMARY KEY AUTO_INCREMENT NOT NULL,
    name VARCHAR(100),
    hostname VARCHAR(100),
    ip VARCHAR(100),
    port INT,
    grpc_port INT,
    prometheus_port INT,
    datapath VARCHAR(500),
    enabled BOOLEAN,
    writeOutput BOOLEAN,
    decode BOOLEAN
);

CREATE TABLE IF NOT EXISTS equipments (
    id INT PRIMARY KEY AUTO_INCREMENT NOT NULL,
    type INT,
    device_ip VARCHAR(100),
    host_ip VARCHAR(100),
    host_port INT,
    enabled BOOLEAN,
    ldcID INT
);

ALTER TABLE equipments ADD FOREIGN KEY (ldcID) REFERENCES ldcs(id);

CREATE TABLE IF NOT EXISTS duckParams (
    filesize INT NOT NULL,
    experiment VARCHAR(100) NOT NULL,
    packetSize INT NOT NULL,
    nPacketsInBuffer INT NOT NULL,
    buffertimeout INT NOT NULL DEFAULT 30,
    equipmentDataCh INT NOT NULL,
    receiveCh INT NOT NULL,
    dataCh INT NOT NULL,
    metricsCh INT NOT NULL,
    tcpConnectionsCh INT NOT NULL,
    decoderCh INT NOT NULL,
    decoderWorkers INT NOT NULL,
    writerCh INT NOT NULL
);

CREATE TABLE IF NOT EXISTS decoderParams (
    ext_trigger INT NOT NULL,
    trg_code_1 INT NOT NULL,
    trg_code_2 INT NOT NULL,
    read_pmts BOOLEAN,
    read_sipms BOOLEAN,
    read_trigger BOOLEAN,
    split_trigger BOOLEAN,
    no_db BOOLEAN,
    discard BOOLEAN,
    host VARCHAR(100) NOT NULL,
    user VARCHAR(100) NOT NULL,
    passwd VARCHAR(100) NOT NULL,
    db_name VARCHAR(100) NOT NULL,
    write_data BOOLEAN,
    use_blosc BOOLEAN,
    blosc_algorithm VARCHAR(100) NOT NULL,
    compression_level INT NOT NULL,
    bit_shuffle VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS rates (
    run INT NOT NULL,
    last_update TIMESTAMP NULL,
    avgByteRate FLOAT NULL,
    avgTriggerRate FLOAT NULL,
    bytes BIGINT UNSIGNED NULL,
    events BIGINT UNSIGNED NULL
);

CREATE TABLE IF NOT EXISTS testDeviceParams (
    equipment_id INT PRIMARY KEY NOT NULL,
    event_rate_ms INT NOT NULL DEFAULT 12,
    packets_per_event INT NOT NULL DEFAULT 85,
    packet_size INT NOT NULL DEFAULT 500,
    error_injection_rate FLOAT NOT NULL DEFAULT 0.0,
    max_events INT NOT NULL DEFAULT 0,
    FOREIGN KEY (equipment_id) REFERENCES equipments(id)
);

CREATE TABLE IF NOT EXISTS simulatorParams (
    equipmentID INT PRIMARY KEY NOT NULL,
    filePath VARCHAR(512) NOT NULL,
    rateHz FLOAT NOT NULL DEFAULT 0.1,
    loopMode BOOLEAN NOT NULL DEFAULT false,
    maxEvents INT NOT NULL DEFAULT 0,
    packetSize INT NOT NULL DEFAULT 7992,
    FOREIGN KEY (equipmentID) REFERENCES equipments(id)
);
