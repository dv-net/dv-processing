package config

import (
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/gcash/bchd/rpcclient"
)

type BitcoinCashBlockchain struct {
	Enabled    bool   `json:"enabled" default:"false"`
	Network    string `yaml:"network" json:"network" validate:"required,oneof=mainnet testnet" default:"mainnet" example:"mainnet / testnet"`
	Attributes struct {
		FeePerByte    int64 `yaml:"fee_per_byte" json:"fee_per_byte" default:"1"`
		MinUTXOAmount int64 `yaml:"min_utxo_amount" json:"min_utxo_amount" default:"0" usage:"min UTXO amount in satoshi"`
	}
	Node struct {
		Address string `usage:"node address"`
		Login   string `yaml:"login" json:"login" secret:"true" usage:"node login"`
		Secret  string `secret:"true"`
		UseTLS  bool   `yaml:"use_tls" json:"use_tls" default:"true" usage:"use TLS for connection"`
	}
}

func (s BitcoinCashBlockchain) Validate() error {
	if !s.Enabled {
		return nil
	}

	if s.Node.Address == "" {
		return fmt.Errorf("bitcoin cash: node address must not be empty")
	}
	if s.Node.Login == "" {
		return fmt.Errorf("bitcoin cash: node login must not be empty")
	}
	if s.Node.Secret == "" {
		return fmt.Errorf("bitcoin cash: node password must not be empty")
	}

	if s.Attributes.FeePerByte < 0 {
		return fmt.Errorf("bitcoin cash: fee per byte must be greater than or equal to 0")
	}

	if s.Attributes.MinUTXOAmount < 0 {
		return fmt.Errorf("bitcoin cash: min UTXO amount must be greater than or equal to 0")
	}

	return nil
}

func (s BitcoinCashBlockchain) ConvertToSDKConfig(_ constants.ProcessingIdentity) bch.Config {
	return bch.Config{
		RPCConfig: &rpcclient.ConnConfig{
			Host:         s.Node.Address,
			HTTPPostMode: true,
			DisableTLS:   !s.Node.UseTLS,
			User:         s.Node.Login,
			Pass:         s.Node.Secret,
		},
	}
}
