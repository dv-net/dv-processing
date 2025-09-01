package tron

import addr "github.com/fbsobreira/gotron-sdk/pkg/address"

func ValidateAddress(address string) bool {
	_, err := addr.Base58ToAddress(address)
	return err == nil
}
