package config

import "github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"

type Blockchain struct {
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
