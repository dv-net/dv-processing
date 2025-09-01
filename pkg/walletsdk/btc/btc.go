package btc

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/rpcclient"
)

type Config struct {
	RPCConfig *rpcclient.ConnConfig
}

type BTC struct {
	node      *rpcclient.Client
	WalletSDK *WalletSDK
}

func NewBTC(conf Config, walletSDK *WalletSDK) (*BTC, error) {
	node, err := rpcclient.New(conf.RPCConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("init btc connection with node: %w", err)
	}

	return &BTC{
		node:      node,
		WalletSDK: walletSDK,
	}, nil
}

// Name returns the service name
func (t *BTC) Name() string { return "btc-service" }

// Node returns the grpc client
func (t *BTC) Node() *rpcclient.Client { return t.node }

// Start
func (t *BTC) Start(_ context.Context) error {
	return nil
}

// Stop
func (t *BTC) Stop(_ context.Context) error {
	if t.node == nil {
		return nil
	}

	t.node.Shutdown()

	return nil
}
