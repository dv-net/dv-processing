-- name: MaxSequence :one
select 
  coalesce(max(sequence),0)::int as sequence
from cold_wallets w
where w.blockchain = $1 and w.owner_id = $2;

-- name: Get :one
select * from cold_wallets where owner_id = $1 and blockchain = $2 and address = $3 LIMIT 1;

-- name: GetAllByOwnerID :many
select * from cold_wallets where owner_id = $1;

-- name: GetByBlockchainAndAddress :one
select * from cold_wallets where blockchain = $1 and address = $2;
