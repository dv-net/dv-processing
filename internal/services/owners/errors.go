package owners

import "errors"

var (
	ErrClientNotFound       = errors.New("client not found")
	ErrExternalIDExists     = errors.New("an owner with such external identifier already exists")
	ErrEmptyOwnerID         = errors.New("empty owner id")
	ErrEmptyWalletAddressID = errors.New("empty wallet address id")
	ErrEmptyOTP             = errors.New("empty otp")
	ErrEmptyOTPSecret       = errors.New("empty otp secret")
	ErrEmptyMnemonic        = errors.New("empty mnemonic")
	ErrEmptyPassPhrase      = errors.New("empty pass phrase")
	ErrTwoFactorDisabled    = errors.New("two factor authentication is disabled")
)
