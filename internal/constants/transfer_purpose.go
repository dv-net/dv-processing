package constants

import "fmt"

type TransferPurpose string

const (
	// TransferPurposeDefault
	//
	// Regular transfer between owner's wallets (hot -> cold/processing, etc.)
	TransferPurposeDefault TransferPurpose = "default"

	// TransferPurposeWithdrawal
	//
	// Withdrawal to an external address. Requires TOTP and is executed immediately.
	TransferPurposeWithdrawal TransferPurpose = "withdrawal"

	// TransferPurposeRefund
	//
	// Refund of an incoming deposit from a hot wallet.
	// Without TOTP it is allowed only back to the deposit sender and is executed after the configured cooldown.
	// With TOTP it is allowed to any external address and is executed immediately.
	TransferPurposeRefund TransferPurpose = "refund"
)

// String returns the transfer purpose as a string
func (p TransferPurpose) String() string { return string(p) }

// Valid checks if the transfer purpose is valid
func (p TransferPurpose) Valid() bool {
	switch p {
	case TransferPurposeDefault, TransferPurposeWithdrawal, TransferPurposeRefund:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (p *TransferPurpose) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*p = TransferPurpose(s)
	case string:
		*p = TransferPurpose(s)
	default:
		return fmt.Errorf("unsupported scan type for TransferPurpose: %T", src)
	}
	return nil
}
