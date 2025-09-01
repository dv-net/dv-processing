-- name: DeleteByID :exec
DELETE FROM webhooks WHERE id=$1;

