package rpccode

import (
	"errors"
	"fmt"
)

type RPCCode uint32

// String returns the string representation of the RPC code.
func (c RPCCode) String() string { return fmt.Sprintf("%d", c) }

const (
	RPCCodeNotEnoughResources   RPCCode = 3000
	RPCCodeAddressIsTaken       RPCCode = 3001
	RPCCodeMaxFeeExceeded       RPCCode = 3002
	RPCCodeServiceUnavailable   RPCCode = 3003
	RPCCodeNotEnoughBalance     RPCCode = 3004
	RPCCodeBlockchainIsDisabled RPCCode = 4000
	RPCCodeAddressEmptyBalance  RPCCode = 4001
)

var RPCCodes = map[RPCCode]error{
	RPCCodeNotEnoughResources:   errors.New("not enough resources"),
	RPCCodeAddressIsTaken:       errors.New("the address is occupied by another transaction"),
	RPCCodeMaxFeeExceeded:       errors.New("max fee value exceeded"),
	RPCCodeBlockchainIsDisabled: errors.New("blockchain is disabled"),
	RPCCodeAddressEmptyBalance:  errors.New("address empty balance"),
	RPCCodeNotEnoughBalance:     errors.New("not enough balance"),
	RPCCodeServiceUnavailable:   errors.New("service unavailable"),
}

func GetErrorByCode(code RPCCode) error {
	err, ok := RPCCodes[code]
	if !ok {
		return fmt.Errorf("error not found for code %d", code)
	}
	return err
}
