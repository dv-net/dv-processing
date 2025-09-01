// Address validator for blockchain wallets and addresses
package avalidator

import (
	"fmt"

	"github.com/dv-net/dv-processing/pkg/chainparams"

	"github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	bchchaincfg "github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	ltcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/ltcutil"
)

// Bitcoin validates a Bitcoin address for Mainnet or Testnet.
func ValidateBitcoinAddress(addr string) bool {
	// Main network
	_, err := btcutil.DecodeAddress(addr, &btcchaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = btcutil.DecodeAddress(addr, &btcchaincfg.TestNet3Params)
	return err == nil
}

// Litecoin validates a Litecoin address for Mainnet or Testnet.
func ValidateLitecoinAddress(addr string) bool {
	// Main network
	_, err := ltcutil.DecodeAddress(addr, &ltcchaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = ltcutil.DecodeAddress(addr, &ltcchaincfg.TestNet4Params)
	return err == nil
}

// BitcoinCash validates a Bitcoin Cash address for Mainnet or Testnet.
func ValidateBitcoinCashAddress(addr string) bool {
	// Main network
	_, err := bchutil.DecodeAddress(addr, &bchchaincfg.MainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = bchutil.DecodeAddress(addr, &bchchaincfg.TestNet3Params)
	if err == nil {
		return true
	}

	_, err = bchutil.DecodeAddress(addr, &bchchaincfg.TestNet4Params)
	return err == nil
}

// Dogecoin validates a Dogecoin address for Mainnet or Testnet.
func ValidateDogecoinAddress(addr string) bool {
	// Main network
	_, err := ltcutil.DecodeAddress(addr, &chainparams.DogecoinMainNetParams)
	if err == nil {
		return true
	}

	// Test network
	_, err = ltcutil.DecodeAddress(addr, &chainparams.DogecoinTestNet3Params)
	return err == nil
}

// EVMAddress validates an EVM-compatible address (Ethereum, BSC, Polygon).
func ValidateEVMAddress(addr string) bool {
	return common.IsHexAddress(addr)
}

// Tron validates a Tron address.
func ValidateTronAddress(addr string) bool {
	_, err := address.Base58ToAddress(addr)
	return err == nil
}

// ValidateAddressByBlockchain validates an address by blockchain type.
func ValidateAddressByBlockchain(addr, blockchain string) bool {
	switch blockchain {
	case BlockchainTypeBitcoin:
		return ValidateBitcoinAddress(addr)
	case BlockchainTypeLitecoin:
		return ValidateLitecoinAddress(addr)
	case BlockchainTypeDogecoin:
		return ValidateDogecoinAddress(addr)
	case BlockchainTypeEthereum,
		BlockchainTypeBinanceSmartChain,
		BlockchainTypePolygon,
		BlockchainTypeArbitrum:
		return ValidateEVMAddress(addr)
	case BlockchainTypeTron:
		return ValidateTronAddress(addr)
	case BlockchainTypeBitcoinCash:
		return ValidateBitcoinCashAddress(addr)
	default:
		return false
	}
}

// ValidateAddressesByBlockchain validates multiple addresses by blockchain type.
func ValidateAddressesByBlockchain(addresses []string, blockchain string) error {
	for _, addr := range addresses {
		if !ValidateAddressByBlockchain(addr, blockchain) {
			return fmt.Errorf("invalid address: %s", addr)
		}
	}
	return nil
}
