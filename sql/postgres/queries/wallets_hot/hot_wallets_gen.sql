-- name: Create :one
INSERT INTO hot_wallets (blockchain, address, owner_id, external_wallet_id, sequence, is_activated, is_active, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, now())
	RETURNING *;

-- name: Exist :one
SELECT EXISTS (SELECT 1 FROM hot_wallets WHERE address=$1 AND blockchain=$2 AND owner_id=$3 LIMIT 1)::boolean;

