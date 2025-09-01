-- name: GetGlobalByName :one
SELECT *
FROM settings
WHERE name = $1
  AND model_id IS NULL
  AND model_type IS NULL
LIMIT 1;

-- name: SetGlobal :one
INSERT INTO settings (name, value, created_at) VALUES ($1, $2, now()) ON CONFLICT (model_id, model_type, name) DO UPDATE set value = $2 returning *;