package bch

import (
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
)

func ValidateAddress(addrress string) bool {
	// Main network
	_, err := bchutil.DecodeAddress(addrress, &chaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = bchutil.DecodeAddress(addrress, &chaincfg.TestNet3Params)
	if err == nil {
		return true
	}

	_, err = bchutil.DecodeAddress(addrress, &chaincfg.TestNet4Params)
	return err == nil
}
