package bch

import (
	"context"
	"fmt"

	"github.com/gcash/bchd/rpcclient"
)

type Config struct {
	RPCConfig *rpcclient.ConnConfig
}

type BCH struct {
	node      *rpcclient.Client
	WalletSDK *WalletSDK
}

func NewBCH(conf Config, walletSDK *WalletSDK) (*BCH, error) {
	node, err := rpcclient.New(conf.RPCConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("init bch connection with node: %w", err)
	}

	return &BCH{
		node:      node,
		WalletSDK: walletSDK,
	}, nil
}

// Name returns the service name
func (t *BCH) Name() string { return "bch-service" }

// Node returns the grpc client
func (t *BCH) Node() *rpcclient.Client { return t.node }

// Start
func (t *BCH) Start(_ context.Context) error {
	return nil
}

// Stop
func (t *BCH) Stop(_ context.Context) error {
	if t.node == nil {
		return nil
	}

	t.node.Shutdown()

	return nil
}
