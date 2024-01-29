-- name: GetLDC :one
SELECT * FROM ldcs WHERE id = ?;

-- name: ListLDCs :many
SELECT * FROM ldcs ORDER BY name;

-- name: CreateLDC :execresult
INSERT INTO ldcs (
    name, hostname, ip, enabled, grpc_port, prometheus_port
) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateLDC :exec
UPDATE ldcs 
SET name = ?, hostname = ?, ip = ?, enabled = ?, 
    grpc_port = ?, prometheus_port = ?
WHERE id = ?;

-- name: DeleteLDC :exec
DELETE FROM ldcs WHERE id = ?;