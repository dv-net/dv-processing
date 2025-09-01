package bch_test

import (
	"fmt"

	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/gcash/bchd/chaincfg"
)

func generateTestSegwitAddresses(count int) ([]*bch.GenerateAddressData, error) {
	wsdk := bch.NewWalletSDK(&chaincfg.MainNetParams)

	addresses := make([]*bch.GenerateAddressData, 0, count)
	for i := range count {
		addrData, err := wsdk.GenerateAddress(mnemonic, passphrase, uint32(i))
		if err != nil {
			return nil, fmt.Errorf("generate test segwit addresses: %w", err)
		}
		addresses = append(addresses, addrData)
	}

	return addresses, nil
}
