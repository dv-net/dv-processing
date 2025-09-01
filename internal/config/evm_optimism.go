package config

import "fmt"

type OptimismBlockchain struct {
	Enabled bool   `json:"enabled" default:"false"`
	Network string `yaml:"network" json:"network" validate:"required,oneof=mainnet testnet" default:"mainnet" example:"mainnet / testnet"`
	Node    struct {
		Address string `json:"address" yaml:"address" usage:"node address"`
	}
	Attributes struct {
		MaxGasPrice float64 `yaml:"max_gas_price" json:"max_gas_price" validate:"gte=0" default:"0.001" usage:"max gas price in Gwei"`
	}
}

// Validate
func (s OptimismBlockchain) Validate() error {
	if !s.Enabled {
		return nil
	}

	if s.Network == "" {
		return fmt.Errorf("network must not be empty")
	}

	if s.Node.Address == "" {
		return fmt.Errorf("node address must not be empty")
	}

	return nil
}

// GetMaxGassFee returns the max gas fee in Gwei
func (s OptimismBlockchain) GetMaxGasFee() float64 {
	return s.Attributes.MaxGasPrice
}

// IsEnabled returns true if the blockchain is enabled
func (s OptimismBlockchain) IsEnabled() bool {
	return s.Enabled
}
