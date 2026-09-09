-- name: GetDuckParams :one
SELECT * FROM duckParams LIMIT 1;

-- name: GetLatestRun :one
SELECT id FROM runs ORDER BY id DESC LIMIT 1;

-- name: GetDecoderParams :one
SELECT ext_trigger, trg_code_1, trg_code_2, read_pmts, read_sipms, read_trigger, read_fibers, split_trigger, no_db, discard, host, user, passwd, db_name, write_data, use_blosc, blosc_algorithm, compression_level, bit_shuffle FROM decoderParams LIMIT 1;

-- name: UpdateDecoderParams :exec
INSERT INTO decoderParams (
    ext_trigger, trg_code_1, trg_code_2, read_pmts, read_sipms,
    read_trigger, read_fibers, split_trigger, no_db, discard, host,
    user, passwd, db_name, write_data, use_blosc,
    blosc_algorithm, compression_level, bit_shuffle
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: TruncateDecoderParams :exec
TRUNCATE TABLE decoderParams;

-- name: GetDisabledGDCs :many
SELECT * FROM gdcs WHERE enabled = 0;

-- name: GetDisabledLDCs :many
SELECT * FROM ldcs WHERE enabled = 0;

-- name: GetDisabledEquipments :many
SELECT * FROM equipments WHERE enabled = 0;

-- name: GetGDCsWithoutWrite :many
SELECT * FROM gdcs WHERE writeOutput = 0;

-- name: GetTestDeviceParams :many
SELECT * FROM testDeviceParams;

-- Run management queries
-- name: GetLatestRunWithTimestamp :one
SELECT id, start, stop FROM runs ORDER BY id DESC LIMIT 1;

-- name: InsertNewRun :exec
INSERT INTO runs (start, stop) VALUES (NULL, NULL);

-- name: UpdateRunStartTime :exec
UPDATE runs SET start = ?, stop = NULL WHERE id = ?;

-- name: UpdateRunStopTime :exec
UPDATE runs SET stop = ? WHERE id = ?;

-- name: InsertGDCEvents :exec
INSERT INTO events (run, gdc_id, ldc_id, events) VALUES (?, ?, NULL, ?);

-- name: InsertGDCBytes :exec
INSERT INTO data (run, gdc_id, ldc_id, bytes) VALUES (?, ?, NULL, ?);

-- name: InsertLDCEvents :exec
INSERT INTO events (run, gdc_id, ldc_id, events) VALUES (?, NULL, ?, ?);

-- name: InsertLDCBytes :exec
INSERT INTO data (run, gdc_id, ldc_id, bytes) VALUES (?, NULL, ?, ?);

-- name: InsertLDCErrorCount :exec
INSERT INTO errors (run, gdc_id, ldc_id, errors) VALUES (?, NULL, ?, ?);

-- name: InsertGDCErrorCount :exec
INSERT INTO errors (run, gdc_id, ldc_id, errors) VALUES (?, ?, NULL, ?);

-- Rate management queries
-- name: GetRateByRun :one
SELECT run FROM rates WHERE run = ?;

-- name: InsertRate :exec
INSERT INTO rates (run, last_update, bytes, events, avgTriggerRate, avgByteRate) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateRate :exec
UPDATE rates SET last_update = ?, bytes = ?, events = ?, avgTriggerRate = ?, avgByteRate = ? WHERE run = ?;

-- name: GetTopiParams :one
SELECT * FROM topiParams WHERE id = 1;

-- name: UpsertTopiParams :exec
INSERT INTO topiParams (
  id, enabled, daemon_url, api_token, rabbitmq_address, rabbitmq_port,
  rabbitmq_user, rabbitmq_password, rabbitmq_vhost, exchange_name,
  control_queue, selected_configuration
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  enabled = VALUES(enabled), daemon_url = VALUES(daemon_url),
  api_token = VALUES(api_token), rabbitmq_address = VALUES(rabbitmq_address),
  rabbitmq_port = VALUES(rabbitmq_port), rabbitmq_user = VALUES(rabbitmq_user),
  rabbitmq_password = VALUES(rabbitmq_password), rabbitmq_vhost = VALUES(rabbitmq_vhost),
  exchange_name = VALUES(exchange_name), control_queue = VALUES(control_queue),
  selected_configuration = VALUES(selected_configuration);
