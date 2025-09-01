package models

import (
	"fmt"
	"time"

	commonv1 "github.com/dv-net/dv-processing/api/processing/common/v1"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/shopspring/decimal"
)

type Asset struct {
	ID     string
	Amount decimal.Decimal
}

type LastProcessedBlock struct {
	Blockchain       wconstants.BlockchainType
	StartBlockNumber uint64
	EndBlockNumber   uint64
}

type BlockchainBlock struct {
	Blockchain wconstants.BlockchainType
	Number     uint64
}

const DefaultWalletTTL = time.Minute * 20

// ConvertBlockchainType converts commonv1.Blockchain to wconstants.BlockchainType.
func ConvertBlockchainType(blockchain commonv1.Blockchain) (wconstants.BlockchainType, error) {
	var res wconstants.BlockchainType
	switch blockchain {
	// Tron
	case commonv1.Blockchain_BLOCKCHAIN_TRON:
		res = wconstants.BlockchainTypeTron

	// EVM
	case commonv1.Blockchain_BLOCKCHAIN_ETHEREUM:
		res = wconstants.BlockchainTypeEthereum
	case commonv1.Blockchain_BLOCKCHAIN_BINANCE_SMART_CHAIN:
		res = wconstants.BlockchainTypeBinanceSmartChain
	case commonv1.Blockchain_BLOCKCHAIN_POLYGON:
		res = wconstants.BlockchainTypePolygon
	case commonv1.Blockchain_BLOCKCHAIN_ARBITRUM:
		res = wconstants.BlockchainTypeArbitrum
	case commonv1.Blockchain_BLOCKCHAIN_OPTIMISM:
		res = wconstants.BlockchainTypeOptimism
	case commonv1.Blockchain_BLOCKCHAIN_LINEA:
		res = wconstants.BlockchainTypeLinea

	// BTC Like
	case commonv1.Blockchain_BLOCKCHAIN_BITCOIN:
		res = wconstants.BlockchainTypeBitcoin
	case commonv1.Blockchain_BLOCKCHAIN_LITECOIN:
		res = wconstants.BlockchainTypeLitecoin
	case commonv1.Blockchain_BLOCKCHAIN_BITCOINCASH:
		res = wconstants.BlockchainTypeBitcoinCash
	case commonv1.Blockchain_BLOCKCHAIN_DOGECOIN:
		res = wconstants.BlockchainTypeDogecoin

	default:
		return res, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	if !res.Valid() {
		return "", fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	return res, nil
}

// ConvertBlockchainTypeToPb
func ConvertBlockchainTypeToPb(blockchain wconstants.BlockchainType) commonv1.Blockchain {
	switch blockchain {
	// Tron
	case wconstants.BlockchainTypeTron:
		return commonv1.Blockchain_BLOCKCHAIN_TRON

	// EVM
	case wconstants.BlockchainTypeEthereum:
		return commonv1.Blockchain_BLOCKCHAIN_ETHEREUM
	case wconstants.BlockchainTypeBinanceSmartChain:
		return commonv1.Blockchain_BLOCKCHAIN_BINANCE_SMART_CHAIN
	case wconstants.BlockchainTypePolygon:
		return commonv1.Blockchain_BLOCKCHAIN_POLYGON
	case wconstants.BlockchainTypeArbitrum:
		return commonv1.Blockchain_BLOCKCHAIN_ARBITRUM
	case wconstants.BlockchainTypeOptimism:
		return commonv1.Blockchain_BLOCKCHAIN_OPTIMISM
	case wconstants.BlockchainTypeLinea:
		return commonv1.Blockchain_BLOCKCHAIN_LINEA

	// BTC Like
	case wconstants.BlockchainTypeBitcoin:
		return commonv1.Blockchain_BLOCKCHAIN_BITCOIN
	case wconstants.BlockchainTypeLitecoin:
		return commonv1.Blockchain_BLOCKCHAIN_LITECOIN
	case wconstants.BlockchainTypeBitcoinCash:
		return commonv1.Blockchain_BLOCKCHAIN_BITCOINCASH
	case wconstants.BlockchainTypeDogecoin:
		return commonv1.Blockchain_BLOCKCHAIN_DOGECOIN

	default:
		return commonv1.Blockchain_BLOCKCHAIN_UNSPECIFIED
	}
}
