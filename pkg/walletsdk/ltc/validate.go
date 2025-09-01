package ltc

import (
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/ltcutil"
)

func ValidateAddress(address string) bool {
	// Main network
	_, err := ltcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = ltcutil.DecodeAddress(address, &chaincfg.TestNet4Params)
	return err == nil
}
