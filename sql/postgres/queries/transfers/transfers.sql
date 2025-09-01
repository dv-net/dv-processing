-- name: SetStatus :exec
update transfers set updated_at = now(), status = $2 where id = $1;

-- name: SetTxHash :one
update transfers set updated_at = now(), tx_hash = $2 where id = $1 returning *;

-- name: GetByTxHashAndOwnerID :one
select * from transfers where tx_hash = $1 and owner_id = $2;

-- name: ExistsByTxHashAndOwnerID :one
select exists(select 1 from transfers where tx_hash = $1 and owner_id = $2);

-- name: GetWorkflowSnapshot :one
SELECT workflow_snapshot FROM transfers WHERE id = $1;

-- name: SetWorkflowSnapshot :exec
UPDATE transfers SET workflow_snapshot = $2, updated_at = now() WHERE id = $1;

-- name: GetStateData :one
SELECT state_data FROM transfers WHERE id = $1;

-- name: SetStateData :exec
UPDATE transfers SET state_data = $2, updated_at = now() WHERE id = $1;

-- name: FindAllNewTransfers :many
select * from transfers where status = 'new' order by created_at asc limit 100;

-- name: GetByRequestID :one
select * from transfers where request_id = $1;

-- name: GetActiveTronTransfersResources :one
with dataset as (
	select * from transfers where blockchain = 'tron' and kind = 'resources' and status in ('new', 'processing', 'unconfirmed')
)
select
	coalesce(sum((state_data->'estimated_resources'->'need_to_delegate'->>'energy')::numeric),0)::numeric energy,
	coalesce(sum((state_data->'estimated_resources'->>'need_bandwidth_from_processing_wallet')::numeric),0)::numeric bandwidth,
	coalesce(sum((state_data->'estimated_activation'->>'energy')::numeric),0)::numeric activation_energy,
	coalesce(sum((state_data->'estimated_activation'->>'bandwidth')::numeric),0)::numeric activation_bandwidth
	from dataset
	where state_data->'estimated_resources'->'need_to_delegate'->'energy' is not null and state_data->'estimated_resources'->'need_bandwidth_from_processing_wallet' is not null and state_data->'estimated_activation'->'energy' is not null and state_data->'estimated_activation'->'bandwidth' is not null;


-- name: GetActiveTronTransfersBurn :one
with dataset as (
	select * from transfers where blockchain = 'tron' and kind = 'burntrx' and status in ('new', 'processing', 'unconfirmed')
)
select
	coalesce(sum((state_data->'estimated_resources'->>'trx')::numeric),0)::numeric trx,
	coalesce(sum((state_data->'estimated_activation'->>'energy')::numeric),0)::numeric activation_energy,
	coalesce(sum((state_data->'estimated_activation'->>'bandwidth')::numeric),0)::numeric activation_bandwidth,
	coalesce(sum((state_data->'estimated_activation'->>'trx')::numeric),0)::numeric activation_trx
	from dataset
	where state_data->'estimated_resources'->'trx' is not null and state_data->'estimated_activation'->'energy' is not null and state_data->'estimated_activation'->'bandwidth' is not null and state_data->'estimated_activation'->'trx' is not null;
