-- name: Create :one
INSERT INTO owners (external_id, client_id, mnemonic, pass_phrase, created_at, otp_data)
	VALUES ($1, $2, $3, $4, now(), $5)
	RETURNING *;

-- name: ExistsByExternalID :one
SELECT EXISTS (SELECT 1 FROM owners WHERE external_id=$1 LIMIT 1)::boolean;

-- name: GetAll :many
SELECT * FROM owners;

-- name: GetByID :one
SELECT * FROM owners WHERE id=$1 LIMIT 1;

