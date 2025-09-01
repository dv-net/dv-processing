-- name: GetManyByOwnerAndWalletIDs :many
select * from hot_wallets where id in (select unnest($1::uuid[])) AND owner_id = $2;

-- name: GetManyByOwnerAndWalletAddresses :many
select * from hot_wallets where address in (select unnest($1::text[])) AND owner_id = $2;

-- name: GetAllByOwnerID :many
select * from hot_wallets where owner_id = $1;

-- name: MarkDirty :exec
update hot_wallets w set is_dirty = true, updated_at = now() where w.blockchain = $1 and w.address = $2 and w.owner_id = $3;

-- name: ActivateWallet :exec
update hot_wallets w set is_activated = true, updated_at = now() where w.blockchain = $1 and w.address = $2 and w.owner_id = $3;

-- name: Get :one
select * from hot_wallets where address = $3 and owner_id = $1 and blockchain = $2 limit 1;

-- name: GetByBlockchainAndAddress :one
select * from hot_wallets where blockchain = $1 and address = $2;

-- name: GetAll :many
select * from hot_wallets where is_active = true;

-- name: FindEVMByExternalID :many
select * from hot_wallets where external_wallet_id = $1 and blockchain in (select unnest($2::text[])) and owner_id = $3 and is_active = true order by sequence desc limit 1;
