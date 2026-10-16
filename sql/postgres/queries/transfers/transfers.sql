-- name: SetStatus :exec
update transfers
set updated_at = now(),
    status     = $2
where id = $1;

-- name: SetTxHash :one
update transfers
set updated_at = now(),
    tx_hash    = $2
where id = $1
returning *;

-- name: GetByTxHashAndOwnerID :one
select *
from transfers
where tx_hash = $1
  and owner_id = $2;

-- name: ExistsByTxHashAndOwnerID :one
select exists(select 1 from transfers where tx_hash = $1 and owner_id = $2);

-- name: GetWorkflowSnapshot :one
SELECT workflow_snapshot
FROM transfers
WHERE id = $1;

-- name: SetWorkflowSnapshot :exec
UPDATE transfers
SET workflow_snapshot = $2,
    updated_at        = now()
WHERE id = $1;

-- name: GetStateData :one
SELECT state_data
FROM transfers
WHERE id = $1;

-- name: SetStateData :exec
UPDATE transfers
SET state_data = $2,
    updated_at = now()
WHERE id = $1;

-- name: FindAllNewTransfers :many
select *
from transfers
where status = 'new'
  and (execute_after is null or execute_after <= now())
order by created_at asc
limit 100;

-- name: GetByRequestID :one
select *
from transfers
where request_id = $1;

-- name: GetActiveTronTransfersResources :one
with dataset as (select *
                 from transfers
                 where blockchain = 'tron'
                   and kind = 'resources'
                   and status in ('new', 'processing', 'unconfirmed')
                   and (execute_after is null or execute_after <= now()))
select coalesce(sum((state_data -> 'estimated_resources' -> 'need_to_delegate' ->> 'energy')::numeric),
                0)::numeric                                                                       energy,
       coalesce(sum((state_data -> 'estimated_resources' ->> 'need_bandwidth_from_processing_wallet')::numeric),
                0)::numeric                                                                       bandwidth,
       coalesce(sum((state_data -> 'estimated_activation' ->> 'energy')::numeric), 0)::numeric    activation_energy,
       coalesce(sum((state_data -> 'estimated_activation' ->> 'bandwidth')::numeric), 0)::numeric activation_bandwidth
from dataset
where state_data -> 'estimated_resources' -> 'need_to_delegate' -> 'energy' is not null
  and state_data -> 'estimated_resources' -> 'need_bandwidth_from_processing_wallet' is not null
  and state_data -> 'estimated_activation' -> 'energy' is not null
  and state_data -> 'estimated_activation' -> 'bandwidth' is not null;


-- name: GetActiveTronTransfersBurn :one
with dataset as (select *
                 from transfers
                 where blockchain = 'tron'
                   and kind = 'burntrx'
                   and status in ('new', 'processing', 'unconfirmed')
                   and (execute_after is null or execute_after <= now()))
select coalesce(sum((state_data -> 'estimated_resources' ->> 'trx')::numeric), 0)::numeric        trx,
       coalesce(sum((state_data -> 'estimated_activation' ->> 'energy')::numeric), 0)::numeric    activation_energy,
       coalesce(sum((state_data -> 'estimated_activation' ->> 'bandwidth')::numeric), 0)::numeric activation_bandwidth,
       coalesce(sum((state_data -> 'estimated_activation' ->> 'trx')::numeric), 0)::numeric       activation_trx
from dataset
where state_data -> 'estimated_resources' -> 'trx' is not null
  and state_data -> 'estimated_activation' -> 'energy' is not null
  and state_data -> 'estimated_activation' -> 'bandwidth' is not null
  and state_data -> 'estimated_activation' -> 'trx' is not null;

-- name: SetProcessingIfNew :execrows
update transfers
set status     = 'processing',
    updated_at = now()
where id = $1
  and status = 'new';

-- name: Cancel :one
update transfers
set status     = 'canceled',
    updated_at = now()
where owner_id = $1
  and request_id = $2
  and status = 'new'
  and execute_after > now()
returning *;

-- name: ExecuteNow :one
update transfers
set execute_after           = now(),
    two_factor_confirmed_at = now(),
    updated_at              = now()
where owner_id = $1
  and request_id = $2
  and status = 'new'
  and execute_after > now()
returning *;

-- name: ExistsActiveRefundBySource :one
select exists(select 1
              from transfers
              where owner_id = $1
                and blockchain = $2
                and source_tx_hash = $3
                and purpose = 'refund'
                and (status not in ('failed', 'canceled') or broadcast_at is not null));

-- name: SetBroadcastAt :exec
update transfers
set broadcast_at = coalesce(broadcast_at, now()),
    updated_at   = now()
where id = $1;

-- name: ClearBroadcastAt :exec
-- the transfer transaction was rejected by the node and is not in the network
update transfers
set broadcast_at = null,
    updated_at   = now()
where id = $1;
