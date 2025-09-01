package btc

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// Bitcoin
func ValidateAddress(addrress string) bool {
	// Main network
	_, err := btcutil.DecodeAddress(addrress, &chaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = btcutil.DecodeAddress(addrress, &chaincfg.TestNet3Params)
	return err == nil
}
