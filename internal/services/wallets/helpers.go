package wallets

import (
	"fmt"

	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

// AddressTypeByBlockchain returns the default address type by blockchain type.
func AddressTypeByBlockchain(blockchain wconstants.BlockchainType) (string, error) {
	if !blockchain.Valid() {
		return "", fmt.Errorf("invalid blockchain type: %s", blockchain)
	}

	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		return string(btc.AddressTypeP2WPKH), nil
	case wconstants.BlockchainTypeLitecoin:
		return string(ltc.AddressTypeP2WPKH), nil
	case wconstants.BlockchainTypeDogecoin:
		return string(doge.AddressTypeP2PKH), nil
	default:
		return "", nil
	}
}
