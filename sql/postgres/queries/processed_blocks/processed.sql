-- name: LastBlockNumber :one
select "number" from processed_blocks pb where pb.blockchain = $1 limit 1;

-- name: UpdateNumber :exec
UPDATE processed_blocks SET "number" = $2, updated_at=now() WHERE blockchain = $1;
