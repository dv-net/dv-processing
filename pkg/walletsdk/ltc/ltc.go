package ltc

import (
	"context"
	"fmt"

	"github.com/ltcsuite/ltcd/rpcclient"
)

type Config struct {
	RPCConfig *rpcclient.ConnConfig
}

type LTC struct {
	node      *rpcclient.Client
	WalletSDK *WalletSDK
}

func NewLTC(conf Config, walletSDK *WalletSDK) (*LTC, error) {
	node, err := rpcclient.New(conf.RPCConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("init litecoin connection with node: %w", err)
	}

	return &LTC{
		node:      node,
		WalletSDK: walletSDK,
	}, nil
}

// Name returns the service name
func (t *LTC) Name() string { return "ltc-service" }

// Node returns the grpc client
func (t *LTC) Node() *rpcclient.Client { return t.node }

// Start
func (t *LTC) Start(_ context.Context) error {
	return nil
}

// Stop
func (t *LTC) Stop(_ context.Context) error {
	if t.node == nil {
		return nil
	}

	t.node.Shutdown()

	return nil
}
