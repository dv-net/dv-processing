package bch_test

import (
	"fmt"
	"testing"

	"github.com/gcash/bchd/rpcclient"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/mx/cfg"
	"github.com/stretchr/testify/require"
)

var (
	mnemonic   = "rubber cotton curious about boss into layer trial hidden under reason deliver visit shield decide pole venue border maze cake tip pulp shift off"
	passphrase = "47bf5455-607f-4f0e-a32e-ffe8a9fdb2d2" //nolint:gosec
)

func initConfig() (*config.Config, error) {
	conf := new(config.Config)
	if err := cfg.Load(conf,
		cfg.WithLoaderConfig(cfg.Config{
			Files:      []string{"../../../config.yaml"},
			SkipFlags:  true,
			MergeFiles: true,
		}),
	); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return conf, nil
}

func initClient(conf *config.Config) (*bch.BCH, error) {
	sdk := walletsdk.New(walletsdk.Config{
		Bitcoin: walletsdk.BitcoinConfig{
			Network: conf.Blockchain.Bitcoin.Network,
		},
		Litecoin: walletsdk.LitecoinConfig{
			Network: conf.Blockchain.Litecoin.Network,
		},
		BitcoinCash: walletsdk.BitcoinCashConfig{
			Network: conf.Blockchain.BitcoinCash.Network,
		},
	})

	c := bch.Config{
		RPCConfig: &rpcclient.ConnConfig{
			Host:         conf.Blockchain.BitcoinCash.Node.Address,
			HTTPPostMode: true,
			DisableTLS:   !conf.Blockchain.BitcoinCash.Node.UseTLS,
			User:         conf.Blockchain.BitcoinCash.Node.Login,
			Pass:         conf.Blockchain.BitcoinCash.Node.Secret,
		},
	}

	cl, err := bch.NewBCH(c, sdk.BCH)
	if err != nil {
		return nil, fmt.Errorf("init btc client: %w", err)
	}

	return cl, nil
}

func TestLastBlock(t *testing.T) {
	conf, err := initConfig()
	require.NoError(t, err)

	btcClient, err := initClient(conf)
	require.NoError(t, err)

	lastBlock, err := btcClient.Node().GetBestBlockHash()
	require.NoError(t, err)

	block, err := btcClient.Node().GetBlock(lastBlock)
	require.NoError(t, err)

	testutils.PrintJSON(block)
}
