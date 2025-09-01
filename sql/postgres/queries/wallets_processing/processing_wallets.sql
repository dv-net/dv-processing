-- name: GetAllByOwnerID :many
select * from processing_wallets where owner_id = $1;

-- name: Get :one
select * from processing_wallets where address = $2 and blockchain = $1 limit 1;

-- name: GetByOwnerID :one
select * from processing_wallets where address = $3 and owner_id = $1 and blockchain = $2 limit 1;

-- name: GetByBlockchain :one
select * from processing_wallets where owner_id = $1 and blockchain = $2 limit 1;

-- name: GetByBlockchainAndAddress :one
select * from processing_wallets where blockchain = $1 and address = $2;

-- name: GetAllNotCreatedWallets :many
WITH all_combinations AS (
  SELECT
    o.id owner_id,
    unnest(@blockchains::varchar[]) AS blockchain
  FROM owners o
),
missing_combinations AS (
  SELECT
    ac.owner_id,
    ac.blockchain::varchar blockchain
  FROM all_combinations ac
  LEFT JOIN processing_wallets w
  ON ac.owner_id = w.owner_id AND ac.blockchain = w.blockchain
  WHERE w.blockchain IS NULL
)
SELECT * FROM missing_combinations;

-- name: IsTakenByAnotherOwner :one
select exists(select 1 from processing_wallets where owner_id != $1 and address = $2);
