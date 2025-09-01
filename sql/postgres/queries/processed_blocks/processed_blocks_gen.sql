-- name: Create :exec
INSERT INTO processed_blocks (blockchain, number, created_at)
	VALUES ($1, $2, now());

