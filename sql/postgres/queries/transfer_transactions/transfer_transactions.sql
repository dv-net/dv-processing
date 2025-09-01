-- name: GetByTransfer :many
SELECT *
FROM transfer_transactions
WHERE transfer_id = $1;

-- name: UpdateStatus :exec
UPDATE transfer_transactions
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: FindTransactionByType :many
SELECT *
FROM transfer_transactions
WHERE transfer_id = $1
  AND tx_type = $2;


-- name: UpdatePendingTxExpense :exec
UPDATE transfer_transactions
SET bandwidth_amount    = sqlc.arg(bandwidth_amount),
    energy_amount       = sqlc.arg(energy_amount),
    native_token_amount = sqlc.arg(native_token_amount),
    native_token_fee    = sqlc.arg(native_token_fee),
    status              = sqlc.arg(current_tx_status),
    updated_at          = now()
WHERE transfer_id = sqlc.arg(transfer_id)
  AND tx_hash = sqlc.arg(tx_hash);

-- name: FindSystemTransactions :many
SELECT tt.*
FROM transfer_transactions tt
         INNER JOIN transfers t ON t.id = tt.transfer_id AND t.owner_id = sqlc.arg(owner_id) AND
                                   t.blockchain = sqlc.arg(blockchain)
WHERE tt.tx_hash = sqlc.arg(tx_hash)
  AND (sqlc.arg(system_tx_types)::VARCHAR[] IS NULL OR tx_type = ANY (sqlc.arg(system_tx_types)::VARCHAR[]));

-- name: GetAllByTransfer :many
SELECT *
FROM transfer_transactions tt
WHERE transfer_id = $1;