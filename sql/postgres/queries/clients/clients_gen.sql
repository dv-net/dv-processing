-- name: Create :one
INSERT INTO clients (secret_key, callback_url, created_at)
	VALUES ($1, $2, now())
	RETURNING *;

-- name: ExistsByID :one
SELECT EXISTS (SELECT 1 FROM clients WHERE id=$1 LIMIT 1)::boolean;

-- name: GetAll :many
SELECT * FROM clients;

-- name: GetByID :one
SELECT * FROM clients WHERE id=$1 LIMIT 1;

