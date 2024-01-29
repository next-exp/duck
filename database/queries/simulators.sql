-- name: GetSimulatorParams :one
SELECT * FROM simulatorParams WHERE equipmentID = ?;

-- name: ListSimulatorParams :many
SELECT * FROM simulatorParams ORDER BY equipmentID;

-- name: CreateSimulatorParams :execresult
INSERT INTO simulatorParams (
    equipmentID, filePath, replayrate, loopmode, maxevents, packetsize
) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateSimulatorParams :exec
UPDATE simulatorParams
SET filePath = ?, replayrate = ?, loopmode = ?, maxevents = ?, packetsize = ?
WHERE equipmentID = ?;

-- name: DeleteSimulatorParams :exec
DELETE FROM simulatorParams WHERE equipmentID = ?;
