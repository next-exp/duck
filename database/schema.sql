CREATE DATABASE duck;
GRANT ALL ON duck.* TO 'next'@'%';

use duck;
create table runs ( 
	id int primary key auto_increment not null,
	start timestamp null,
	stop timestamp null);

create table events (
     run int not null,
     gdc_id int null,
     ldc_id int null,
     events BIGINT UNSIGNED);

create table data (
     run int not null,
     gdc_id int null,
     ldc_id int null,
     bytes BIGINT UNSIGNED);

create table errors (
     run int not null,
     gdc_id int null,
     ldc_id int null,
     errors BIGINT UNSIGNED);

create table ldcs ( 
	id int primary key auto_increment not null,
     name VARCHAR(100), 
     hostname VARCHAR(100), 
     ip VARCHAR(100),
     grpc_port int,
     prometheus_port int,
     enabled BOOLEAN);

create table gdcs ( 
	id int primary key auto_increment not null,
     name VARCHAR(100), 
     hostname VARCHAR(100), 
     ip VARCHAR(100), 
     port int,
     grpc_port int,
     prometheus_port int,
     datapath VARCHAR(500),
     enabled BOOLEAN,
     writeOutput BOOLEAN,
     decode BOOLEAN);

create table equipments ( 
	id int primary key auto_increment not null,
     type int, 
     device_ip VARCHAR(100), 
     host_ip VARCHAR(100), 
     host_port int, 
     enabled BOOLEAN,
	ldcID int);
ALTER TABLE equipments ADD FOREIGN KEY (ldcID) REFERENCES ldcs(id);

create table duckParams (
     filesize int not null,
     experiment VARCHAR(100) not null,
     packetSize int not null,
     nPacketsInBuffer int not null,
     buffertimeout int not null,
     equipmentDataCh int not null,
     receiveCh int not null,
     dataCh int not null,
     metricsCh int not null,
     tcpConnectionsCh int not null,
     decoderCh int not null,
     decoderWorkers int not null,
     writerCh int not null);

create table decoderParams (
     ext_trigger int not null,
     trg_code_1 int not null,
     trg_code_2 int not null,
     read_pmts boolean,
     read_sipms boolean,
     read_trigger boolean,
     read_fibers boolean,
     split_trigger boolean,
     no_db boolean,
     discard boolean,
     host VARCHAR(100) not null,
     user VARCHAR(100) not null,
     passwd VARCHAR(100) not null,
     db_name VARCHAR(100) not null,
     write_data boolean,
     use_blosc boolean,
     blosc_algorithm VARCHAR(100) not null,
     compression_level int not null,
     bit_shuffle VARCHAR(100) not null);

create table rates (
     run int not null,
	last_update timestamp null,
     avgByteRate float null,
     avgTriggerRate float null,
     bytes BIGINT UNSIGNED null,
     events BIGINT UNSIGNED null);

create table testDeviceParams (
     equipment_id int not null,
     event_rate_ms int not null,
     packets_per_event int not null,
     packet_size int not null,
     error_injection_rate float not null,
     max_events int not null
);

create table simulatorParams (
     equipmentID int primary key not null,
     filePath VARCHAR(512) not null,
     replayRate float not null default 1.0,
     loopMode boolean not null default false,
     maxEvents int not null default 0,
     packetSize int not null default 500
);
ALTER TABLE simulatorParams ADD FOREIGN KEY (equipmentID) REFERENCES equipments(id);

create table topiParams (
     id int primary key not null,
     enabled boolean not null default false,
     daemon_url VARCHAR(512) not null default '',
     api_token TEXT not null,
     rabbitmq_address VARCHAR(255) not null default '',
     rabbitmq_port int not null default 5672,
     rabbitmq_user VARCHAR(255) not null default '',
     rabbitmq_password TEXT not null,
     rabbitmq_vhost VARCHAR(255) not null default '/',
     exchange_name VARCHAR(255) not null default 'production',
     control_queue VARCHAR(255) not null default 'topi_daemon_control',
     selected_configuration VARCHAR(255) not null default ''
);
