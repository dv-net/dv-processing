-- name: Create :one
INSERT INTO cold_wallets (blockchain, address, owner_id, is_active, created_at)
	VALUES ($1, $2, $3, $4, now())
	RETURNING *;

