//nolint:dupl
package config

import (
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/ltcsuite/ltcd/rpcclient"
)

type LitecoinBlockchain struct {
	Enabled    bool   `json:"enabled" default:"false"`
	Network    string `yaml:"network" json:"network" validate:"required,oneof=mainnet testnet" default:"mainnet" example:"mainnet / testnet"`
	Attributes struct {
		FeePerByte    int64 `yaml:"fee_per_byte" json:"fee_per_byte" default:"10"`
		MinUTXOAmount int64 `yaml:"min_utxo_amount" json:"min_utxo_amount" default:"0" usage:"min UTXO amount in satoshi"`
	}
	Node struct {
		Address string `usage:"node address"`
		Login   string `yaml:"login" json:"login" secret:"" usage:"node login"`
		Secret  string `secret:""`
		UseTLS  bool   `yaml:"use_tls" json:"use_tls" default:"true" usage:"use TLS for connection"`
	}
}

func (s LitecoinBlockchain) Validate() error {
	if !s.Enabled {
		return nil
	}

	if s.Node.Address == "" {
		return fmt.Errorf("litecoin: node address must not be empty")
	}
	if s.Node.Login == "" {
		return fmt.Errorf("litecoin: node login must not be empty")
	}
	if s.Node.Secret == "" {
		return fmt.Errorf("litecoin: node password must not be empty")
	}

	if s.Attributes.FeePerByte < 0 {
		return fmt.Errorf("litecoin: fee per byte must be greater than or equal to 0")
	}

	if s.Attributes.MinUTXOAmount < 0 {
		return fmt.Errorf("litecoin: min UTXO amount must be greater than or equal to 0")
	}

	return nil
}

func (s LitecoinBlockchain) ConvertToSDKConfig(identity constants.ProcessingIdentity) ltc.Config {
	return ltc.Config{
		RPCConfig: &rpcclient.ConnConfig{
			ExtraHeaders: map[string]string{
				constants.ProcessingIDParamName.String():      identity.ID,
				constants.ProcessingVersionParamName.String(): identity.Version,
			},
			Host:         s.Node.Address,
			HTTPPostMode: true,
			DisableTLS:   !s.Node.UseTLS,
			User:         s.Node.Login,
			Pass:         s.Node.Secret,
		},
	}
}
