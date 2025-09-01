package eproxy

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	commonv2 "github.com/dv-net/dv-proto/gen/go/eproxy/common/v2"
)

// ConvertBlockchain converts wconstants.BlockchainType to commonv1.Blockchain
func ConvertBlockchain(bc wconstants.BlockchainType) commonv2.Blockchain {
	switch bc {
	// Tron
	case wconstants.BlockchainTypeTron:
		return commonv2.Blockchain_BLOCKCHAIN_TRON

	// EVM
	case wconstants.BlockchainTypeEthereum:
		return commonv2.Blockchain_BLOCKCHAIN_ETHEREUM
	case wconstants.BlockchainTypeBinanceSmartChain:
		return commonv2.Blockchain_BLOCKCHAIN_BINANCE_SMART_CHAIN
	case wconstants.BlockchainTypePolygon:
		return commonv2.Blockchain_BLOCKCHAIN_POLYGON
	case wconstants.BlockchainTypeArbitrum:
		return commonv2.Blockchain_BLOCKCHAIN_ARBITRUM
	case wconstants.BlockchainTypeOptimism:
		return commonv2.Blockchain_BLOCKCHAIN_OPTIMISM
	case wconstants.BlockchainTypeLinea:
		return commonv2.Blockchain_BLOCKCHAIN_LINEA

	// BTC Like
	case wconstants.BlockchainTypeBitcoin:
		return commonv2.Blockchain_BLOCKCHAIN_BITCOIN
	case wconstants.BlockchainTypeLitecoin:
		return commonv2.Blockchain_BLOCKCHAIN_LITECOIN
	case wconstants.BlockchainTypeBitcoinCash:
		return commonv2.Blockchain_BLOCKCHAIN_BITCOINCASH
	case wconstants.BlockchainTypeDogecoin:
		return commonv2.Blockchain_BLOCKCHAIN_DOGECOIN

	default:
		return commonv2.Blockchain_BLOCKCHAIN_UNSPECIFIED
	}
}
