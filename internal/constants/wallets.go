package constants

import "fmt"

type WalletType string

const (
	WalletTypeCold       WalletType = "cold"
	WalletTypeHot        WalletType = "hot"
	WalletTypeProcessing WalletType = "processing"
)

// String returns the wallet type as a string
func (w WalletType) String() string { return string(w) }

// Valid checks if the wallet type is valid
func (w WalletType) Valid() bool {
	switch w {
	case WalletTypeCold, WalletTypeHot, WalletTypeProcessing:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (w *WalletType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*w = WalletType(s)
	case string:
		*w = WalletType(s)
	default:
		return fmt.Errorf("unsupported scan type for WalletType: %T", src)
	}
	return nil
}
