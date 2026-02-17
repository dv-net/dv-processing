-- name: IsIncidentProcessed :one
SELECT EXISTS(
    SELECT 1
    FROM processed_incidents
    WHERE blockchain = $1 AND id = $2
) AS exists;

-- name: MarkIncidentAsProcessing :exec
INSERT INTO processed_incidents (
    id,
    blockchain,
    incident_type,
    status,
    rollback_from_block,
    rollback_to_block,
    started_at
)
VALUES ($1, $2, $3, 'processing', $4, $5, NOW())
ON CONFLICT (id) DO UPDATE
SET status = 'processing',
    started_at = NOW();

-- name: MarkIncidentAsCompleted :exec
UPDATE processed_incidents
SET status = 'completed',
    completed_at = NOW(),
    error_message = NULL
WHERE blockchain = $1 AND id = $2;

-- name: MarkIncidentAsFailed :exec
UPDATE processed_incidents
SET status = 'failed',
    completed_at = NOW(),
    error_message = $3
WHERE blockchain = $1 AND id = $2;

-- name: GetIncompleteIncidents :many
SELECT *
FROM processed_incidents
WHERE blockchain = $1
  AND status = 'processing'
ORDER BY started_at ASC;

-- name: CleanupOldIncidents :exec
DELETE FROM processed_incidents
WHERE created_at < NOW() - INTERVAL '30 days'
  AND status = 'completed';
