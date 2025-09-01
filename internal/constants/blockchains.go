package constants

import (
	"time"

	commonv1 "github.com/dv-net/dv-processing/api/processing/common/v1"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func BlockchainTypeToPB(bt wconstants.BlockchainType) commonv1.Blockchain {
	switch bt {
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

var minConfirmations = map[wconstants.BlockchainType]uint64{
	// Tron
	wconstants.BlockchainTypeTron: 19,

	// EVM
	wconstants.BlockchainTypeEthereum:          12,
	wconstants.BlockchainTypeBinanceSmartChain: 20,
	wconstants.BlockchainTypePolygon:           12,
	wconstants.BlockchainTypeArbitrum:          20,
	wconstants.BlockchainTypeOptimism:          20,
	wconstants.BlockchainTypeLinea:             20,

	// BTC Like
	wconstants.BlockchainTypeBitcoin:     1,
	wconstants.BlockchainTypeLitecoin:    1,
	wconstants.BlockchainTypeBitcoinCash: 1,
	wconstants.BlockchainTypeDogecoin:    1,
}

var releaseBlockTime = map[wconstants.BlockchainType]time.Duration{
	wconstants.BlockchainTypeTron:              3 * time.Second,
	wconstants.BlockchainTypeEthereum:          12 * time.Second,
	wconstants.BlockchainTypeBinanceSmartChain: 750 * time.Millisecond,
	wconstants.BlockchainTypePolygon:           2 * time.Second,
	wconstants.BlockchainTypeArbitrum:          250 * time.Millisecond,
	wconstants.BlockchainTypeOptimism:          2 * time.Second,
	wconstants.BlockchainTypeLinea:             2*time.Second + time.Millisecond*500,
}

func GetMinConfirmations(blockchain wconstants.BlockchainType) uint64 {
	mc, ok := minConfirmations[blockchain]
	if ok {
		return mc
	}

	return 1
}

func ConfirmationsTimeout(blockchain wconstants.BlockchainType, confirmations uint64) time.Duration {
	requiredConfirmations := GetMinConfirmations(blockchain)
	leftConfirmations := int64(requiredConfirmations - confirmations) //nolint:gosec

	if leftConfirmations > 0 {
		delay := time.Duration(leftConfirmations) * 10 * time.Second
		if bcReleaseTime, ok := releaseBlockTime[blockchain]; ok {
			delay = bcReleaseTime*time.Duration(leftConfirmations) + time.Second*2
		}
		return delay
	}

	return 0
}

func ConfirmationsTimeoutWithRequired(blockchain wconstants.BlockchainType, requiredConfirmations, currentConfirmations uint64) time.Duration {
	leftConfirmations := int64(requiredConfirmations - currentConfirmations) //nolint:gosec

	if leftConfirmations > 0 {
		delay := time.Duration(leftConfirmations) * 10 * time.Second
		if bcReleaseTime, ok := releaseBlockTime[blockchain]; ok {
			delay = bcReleaseTime*time.Duration(leftConfirmations) + time.Second*2
		}
		return delay
	}

	return 0
}
