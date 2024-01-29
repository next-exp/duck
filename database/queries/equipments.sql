-- name: GetEquipment :one
SELECT * FROM equipments WHERE id = ?;

-- name: ListEquipments :many
SELECT * FROM equipments ORDER BY id;

-- name: CreateEquipment :execresult
INSERT INTO equipments (
    type, device_ip, host_ip, host_port, enabled, ldcID
) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateEquipment :exec
UPDATE equipments 
SET type = ?, device_ip = ?, host_ip = ?, host_port = ?, 
    enabled = ?, ldcID = ?
WHERE id = ?;

-- name: DeleteEquipment :exec
DELETE FROM equipments WHERE id = ?;