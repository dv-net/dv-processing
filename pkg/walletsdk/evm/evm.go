package evm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/dv-net/dv-processing/pkg/walletsdk/evm/erc20"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Config struct {
	NodeAddr   string
	RPCOptions []rpc.ClientOption
	Blockchain wconstants.BlockchainType
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	if c.NodeAddr == "" {
		return errors.New("node address is required")
	}

	if !c.Blockchain.Valid() {
		return fmt.Errorf("invalid blockchain type: %s", c.Blockchain)
	}

	if !c.Blockchain.IsEVM() {
		return fmt.Errorf("blockchain type %s is not EVM", c.Blockchain)
	}

	if c.Blockchain.GetAssetIdentifier() == "" {
		return fmt.Errorf("blockchain type %s is EVM, but asset identifier is empty", c.Blockchain)
	}

	return nil
}

type EVM struct {
	node   *ethclient.Client
	abi    abi.ABI
	config Config
}

func NewEVM(ctx context.Context, conf Config) (*EVM, error) {
	cl, err := rpc.DialOptions(ctx, conf.NodeAddr, conf.RPCOptions...)
	if err != nil {
		return nil, fmt.Errorf("prepare client: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20.ERC20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse erc20 abi: %w", err)
	}

	return &EVM{
		node:   ethclient.NewClient(cl),
		abi:    parsedABI,
		config: conf,
	}, nil
}

// Name returns the service name
func (s *EVM) Name() string { return s.config.Blockchain.GetAssetIdentifier() + "-service" }

// Blockchain returns the blockchain type
func (s *EVM) Blockchain() wconstants.BlockchainType { return s.config.Blockchain }

// Node returns the grpc client
func (s *EVM) Node() *ethclient.Client { return s.node }

// Start
func (s *EVM) Start(_ context.Context) error { return nil }

// Stop
func (s *EVM) Stop(_ context.Context) error {
	if s.node == nil {
		return nil
	}

	s.node.Close()

	return nil
}
