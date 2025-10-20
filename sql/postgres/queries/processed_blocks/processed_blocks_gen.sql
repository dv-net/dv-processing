-- name: Create :exec
INSERT INTO processed_blocks (blockchain, number, created_at, hash)
	VALUES ($1, $2, now(), $3);

