-- name: GetGDC :one
SELECT * FROM gdcs WHERE id = ?;

-- name: ListGDCs :many
SELECT * FROM gdcs ORDER BY name;

-- name: CreateGDC :execresult
INSERT INTO gdcs (
    name, hostname, ip, port, datapath, enabled, 
    writeOutput, decode, grpc_port, prometheus_port
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateGDC :exec
UPDATE gdcs 
SET name = ?, hostname = ?, ip = ?, port = ?, 
    datapath = ?, enabled = ?, writeOutput = ?, 
    decode = ?, grpc_port = ?, prometheus_port = ?
WHERE id = ?;

-- name: DeleteGDC :exec
DELETE FROM gdcs WHERE id = ?;