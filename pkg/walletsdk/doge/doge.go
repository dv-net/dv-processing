package doge

import (
	"context"
	"fmt"

	"github.com/ltcsuite/ltcd/rpcclient"
)

type Config struct {
	RPCConfig *rpcclient.ConnConfig
}

type Doge struct {
	node      *rpcclient.Client
	WalletSDK *WalletSDK
}

func NewDoge(conf Config, walletSDK *WalletSDK) (*Doge, error) {
	node, err := rpcclient.New(conf.RPCConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("init Dogecoin connection with node: %w", err)
	}

	return &Doge{
		node:      node,
		WalletSDK: walletSDK,
	}, nil
}

// Name returns the service name
func (d *Doge) Name() string { return "doge-service" }

// Node returns the RPC client
func (d *Doge) Node() *rpcclient.Client { return d.node }

// Start
func (d *Doge) Start(_ context.Context) error {
	return nil
}

// Stop
func (d *Doge) Stop(_ context.Context) error {
	if d.node == nil {
		return nil
	}

	d.node.Shutdown()
	return nil
}
