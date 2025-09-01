package blockchains

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/doge"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/ethereum/go-ethereum/rpc"
)

type Blockchains struct {
	// Tron
	Tron *tron.Tron

	// EVM blockchains
	Ethereum          *evm.EVM
	BinanceSmartChain *evm.EVM
	Polygon           *evm.EVM
	Arbitrum          *evm.EVM
	Optimism          *evm.EVM
	Linea             *evm.EVM

	// BTC Like blockchains
	Bitcoin     *btc.BTC
	Litecoin    *ltc.LTC
	BitcoinCash *bch.BCH
	Dogecoin    *doge.Doge
}

func New(ctx context.Context, conf config.Blockchain, sdk *walletsdk.SDK) (*Blockchains, error) {
	identity, err := constants.IdentityFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get processing identity: %w", err)
	}

	bc := new(Blockchains)

	// init bitcoin
	if conf.Bitcoin.Enabled {
		bc.Bitcoin, err = btc.NewBTC(
			conf.Bitcoin.ConvertToSDKConfig(identity),
			btc.NewWalletSDK(sdk.BTC.ChainParams()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to init bitcoin service: %w", err)
		}
	}

	// init litecoin
	if conf.Litecoin.Enabled {
		bc.Litecoin, err = ltc.NewLTC(
			conf.Litecoin.ConvertToSDKConfig(identity),
			ltc.NewWalletSDK(sdk.LTC.ChainParams()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to init litecoin service: %w", err)
		}
	}

	// init bitcoin cash
	if conf.BitcoinCash.Enabled {
		bc.BitcoinCash, err = bch.NewBCH(conf.BitcoinCash.ConvertToSDKConfig(identity), bch.NewWalletSDK(sdk.BCH.ChainParams()))
		if err != nil {
			return nil, fmt.Errorf("failed to init bitcoin cash service: %w", err)
		}
	}

	// init Dogecoin
	if conf.Dogecoin.Enabled {
		bc.Dogecoin, err = doge.NewDoge(conf.Dogecoin.ConvertToSDKConfig(identity), doge.NewWalletSDK(sdk.Doge.ChainParams()))
		if err != nil {
			return nil, fmt.Errorf("failed to init bitcoin cash service: %w", err)
		}
	}

	// init tron
	if conf.Tron.Enabled {
		bc.Tron, err = tron.NewTron(conf.Tron.ConvertToSDKConfig(identity))
		if err != nil {
			return nil, fmt.Errorf("failed to init tron service: %w", err)
		}
	}

	// init ethereum
	if conf.Ethereum.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)

		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}

		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.Ethereum.Node.Address,
			Blockchain: wconstants.BlockchainTypeEthereum,
		}

		bc.Ethereum, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init eth service: %w", err)
		}
	}

	// init binance smart chain
	if conf.BinanceSmartChain.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)
		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}
		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.BinanceSmartChain.Node.Address,
			Blockchain: wconstants.BlockchainTypeBinanceSmartChain,
		}
		bc.BinanceSmartChain, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init bsc service: %w", err)
		}
	}

	// init polygon
	if conf.Polygon.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)
		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}
		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.Polygon.Node.Address,
			Blockchain: wconstants.BlockchainTypePolygon,
		}
		bc.Polygon, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init polygon service: %w", err)
		}
	}

	// init arbitrum
	if conf.Arbitrum.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)
		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}
		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.Arbitrum.Node.Address,
			Blockchain: wconstants.BlockchainTypeArbitrum,
		}
		bc.Arbitrum, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init arbitrum service: %w", err)
		}
	}

	// init optimism
	if conf.Optimism.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)
		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}
		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.Optimism.Node.Address,
			Blockchain: wconstants.BlockchainTypeOptimism,
		}
		bc.Optimism, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init optimism service: %w", err)
		}
	}

	// init linea
	if conf.Linea.Enabled {
		customHeaders := http.Header{}
		customHeaders.Add(constants.ProcessingIDParamName.String(), identity.ID)
		customHeaders.Add(constants.ProcessingVersionParamName.String(), identity.Version)
		opts := []rpc.ClientOption{
			rpc.WithHeaders(customHeaders),
		}
		c := evm.Config{
			RPCOptions: opts,
			NodeAddr:   conf.Linea.Node.Address,
			Blockchain: wconstants.BlockchainTypeLinea,
		}
		bc.Linea, err = evm.NewEVM(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to init linea service: %w", err)
		}
	}

	return bc, nil
}

func (b *Blockchains) GetEVMByBlockchain(blockchainType wconstants.BlockchainType) (*evm.EVM, error) {
	switch blockchainType {
	case wconstants.BlockchainTypeEthereum:
		return b.Ethereum, nil
	case wconstants.BlockchainTypeBinanceSmartChain:
		return b.BinanceSmartChain, nil
	case wconstants.BlockchainTypePolygon:
		return b.Polygon, nil
	case wconstants.BlockchainTypeArbitrum:
		return b.Arbitrum, nil
	case wconstants.BlockchainTypeOptimism:
		return b.Optimism, nil
	case wconstants.BlockchainTypeLinea:
		return b.Linea, nil
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainType.String())
	}
}
