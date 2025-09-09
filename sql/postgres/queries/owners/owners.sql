-- name: ConfirmTwoFactorAuth :exec
update owners set otp_confirmed = true where id = $1;

-- name: DisableTwoFactorAuth :exec
update owners set otp_secret=$1, otp_confirmed=false where id=$2;

-- name: SetOTPSecret :exec
update owners set otp_secret=$2 where id=$1;

-- name: UpdateMnemonic :exec
update owners set mnemonic = $2 where id = $1;

-- name: SetOTPData :exec
update owners set otp_data = $2 where id = $1;