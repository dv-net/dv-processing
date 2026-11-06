package config

import (
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

type EScanner struct {
	BlocksInChunk int `yaml:"blocks_in_chunk" default:"20"`
}

type Blockchain struct {
	EScanner EScanner `yaml:"e_scanner"`

	// Tron
	Tron TronBlockchain

	// EVM
	Ethereum          EthereumBlockchain
	BinanceSmartChain BinanceSmartChainBlockchain `yaml:"bsc"` //nolint:tagliatelle
	Polygon           PolygonBlockchain
	Arbitrum          ArbitrumBlockchain
	Optimism          OptimismBlockchain
	Linea             LineaBlockchain

	// BTC Like
	Bitcoin     BitcoinBlockchain
	Litecoin    LitecoinBlockchain
	BitcoinCash BitcoinCashBlockchain `yaml:"bitcoin_cash"`
	Dogecoin    DogecoinBlockchain    `yaml:"dogecoin"`
}

// BackfillThreshold returns the scanner jump threshold for the blockchain,
// 0 when the jump is disabled for it.
func (b Blockchain) BackfillThreshold(blockchain wconstants.BlockchainType) int64 {
	switch blockchain {
	case wconstants.BlockchainTypeTron:
		return b.Tron.BackfillThresholdBlocks

	case wconstants.BlockchainTypeEthereum:
		return b.Ethereum.BackfillThresholdBlocks
	case wconstants.BlockchainTypeBinanceSmartChain:
		return b.BinanceSmartChain.BackfillThresholdBlocks
	case wconstants.BlockchainTypePolygon:
		return b.Polygon.BackfillThresholdBlocks
	case wconstants.BlockchainTypeArbitrum:
		return b.Arbitrum.BackfillThresholdBlocks
	case wconstants.BlockchainTypeOptimism:
		return b.Optimism.BackfillThresholdBlocks
	case wconstants.BlockchainTypeLinea:
		return b.Linea.BackfillThresholdBlocks

	case wconstants.BlockchainTypeBitcoin:
		return b.Bitcoin.BackfillThresholdBlocks
	case wconstants.BlockchainTypeLitecoin:
		return b.Litecoin.BackfillThresholdBlocks
	case wconstants.BlockchainTypeBitcoinCash:
		return b.BitcoinCash.BackfillThresholdBlocks
	case wconstants.BlockchainTypeDogecoin:
		return b.Dogecoin.BackfillThresholdBlocks
	}

	return 0
}

// Available blockchains
func (b Blockchain) Available() []wconstants.BlockchainType {
	var available []wconstants.BlockchainType

	// Tron blockchain
	if b.Tron.Enabled {
		available = append(available, wconstants.BlockchainTypeTron)
	}

	// EVM blockchains

	if b.Ethereum.Enabled {
		available = append(available, wconstants.BlockchainTypeEthereum)
	}

	if b.BinanceSmartChain.Enabled {
		available = append(available, wconstants.BlockchainTypeBinanceSmartChain)
	}

	if b.Polygon.Enabled {
		available = append(available, wconstants.BlockchainTypePolygon)
	}

	if b.Arbitrum.Enabled {
		available = append(available, wconstants.BlockchainTypeArbitrum)
	}

	if b.Optimism.Enabled {
		available = append(available, wconstants.BlockchainTypeOptimism)
	}

	if b.Linea.Enabled {
		available = append(available, wconstants.BlockchainTypeLinea)
	}

	// BTC Like blockchains

	if b.Bitcoin.Enabled {
		available = append(available, wconstants.BlockchainTypeBitcoin)
	}

	if b.Litecoin.Enabled {
		available = append(available, wconstants.BlockchainTypeLitecoin)
	}

	if b.BitcoinCash.Enabled {
		available = append(available, wconstants.BlockchainTypeBitcoinCash)
	}

	if b.Dogecoin.Enabled {
		available = append(available, wconstants.BlockchainTypeDogecoin)
	}

	return available
}
