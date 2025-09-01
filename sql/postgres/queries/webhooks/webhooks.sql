-- name: Create :batchexec
INSERT INTO webhooks (kind, status, payload, client_id, created_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT ((payload::text), client_id)
DO UPDATE SET
  updated_at = now(),
  status = 'new',
  attempts = 0,
  response = null,
  sent_at = null;
  
-- name: IncrementAttempt :exec
update webhooks set updated_at = now(), attempts = attempts + 1, response = $2 where id = $1;

-- name: GetUnsent :many
select * from webhook_view w
where
  w.status = $1
  and (
  -- new
    w.updated_at is null
  -- once a minute the first day
    or (extract(epoch from (w.updated_at - w.created_at)) < 3600 * 24)
  -- once an hour the first week
    or (
        extract(epoch from (w.updated_at - w.created_at)) between 3600 * 24 and 3600 * 24 * 7
      and
        extract(epoch from (now() - date_trunc('hour', w.updated_at))) > 3601
    )
  -- once a day
    or (
        extract(epoch from (w.updated_at - w.created_at)) > 3600 * 24 * 7
      and
        date_trunc('day', now()) > date_trunc('day', w.updated_at)
    )
)
order by w.created_at asc
limit sqlc.arg('limit');

-- name: GetByID :one
select * from webhook_view w where w.id = $1 limit 1;

-- name: Exists :one
SELECT EXISTS (SELECT 1 FROM webhooks WHERE payload->>'hash'=$1 AND payload->>'type'=$2)::boolean;

-- name: SetSentAtNow :exec
update webhooks set sent_at=now(), response=$2, status='sent', attempts = attempts + 1, updated_at=now() where id=$1;

-- name: Cleanup :execrows
DELETE FROM webhooks WHERE created_at < $1 AND status = 'sent';
