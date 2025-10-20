-- name: LastBlockNumber :one
SELECT "number" FROM processed_blocks pb WHERE pb.blockchain = $1 LIMIT 1;

-- name: LastBlock :one
SELECT * FROM processed_blocks pb WHERE pb.blockchain = $1 LIMIT 1;

-- name: UpdateNumber :exec
UPDATE processed_blocks SET "number" = $2, updated_at=now() WHERE blockchain = $1;

-- name: UpdateNumberWithHash :exec
UPDATE processed_blocks SET "number" = $2, hash = $3, updated_at=now() WHERE blockchain = $1;
