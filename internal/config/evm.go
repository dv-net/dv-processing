package config

import (
	"fmt"

	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

type IEVMConfig interface {
	GetMaxGasFee() float64
	IsEnabled() bool
}

func (s Blockchain) GetEVMByBlockchainType(blockchainType wconstants.BlockchainType) (IEVMConfig, error) {
	switch blockchainType {
	case wconstants.BlockchainTypeEthereum:
		return s.Ethereum, nil
	case wconstants.BlockchainTypeBinanceSmartChain:
		return s.BinanceSmartChain, nil
	case wconstants.BlockchainTypePolygon:
		return s.Polygon, nil
	case wconstants.BlockchainTypeArbitrum:
		return s.Arbitrum, nil
	case wconstants.BlockchainTypeOptimism:
		return s.Optimism, nil
	case wconstants.BlockchainTypeLinea:
		return s.Linea, nil
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainType)
	}
}
