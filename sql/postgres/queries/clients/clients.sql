-- name: ChangeCallbackURL :exec
update clients set callback_url = $1, updated_at = now() where id = $2;
