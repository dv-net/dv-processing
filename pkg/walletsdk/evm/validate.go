package evm

import "github.com/ethereum/go-ethereum/common"

func ValidateAddress(address string) bool {
	return common.IsHexAddress(address)
}
