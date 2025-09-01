-- name: Create :one
INSERT INTO processing_wallets (blockchain, address, owner_id, sequence, is_active, created_at)
	VALUES ($1, $2, $3, $4, $5, now())
	RETURNING *;

