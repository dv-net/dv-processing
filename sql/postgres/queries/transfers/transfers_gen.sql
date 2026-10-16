-- name: Create :one
INSERT INTO transfers (status, client_id, owner_id, request_id, blockchain, from_addresses, to_addresses, wallet_from_type, asset_identifier, kind, whole_amount, amount, fee, fee_max, tx_hash, created_at, state_data, workflow_snapshot, purpose, source_tx_hash, wallet_to_type, execute_after, two_factor_confirmed_at, broadcast_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, now(), $16, $17, $18, $19, $20, $21, $22, $23)
	RETURNING *;

-- name: GetByID :one
SELECT * FROM transfers WHERE id=$1 LIMIT 1;
