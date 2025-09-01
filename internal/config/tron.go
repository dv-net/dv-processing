package config

import (
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"google.golang.org/grpc"
)

type TronNode struct {
	GrpcAddress string `json:"grpc_address" yaml:"grpc_address" usage:"node address"`
	UseTLS      bool   `json:"use_tls" yaml:"use_tls" usage:"use tls" default:"false"`
}

type TronBlockchain struct {
	Enabled                   bool `json:"enabled" default:"false"`
	Node                      TronNode
	ActivationContractAddress string `json:"activation_contract_address" yaml:"activation_contract_address" default:"TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY"`
	UseBurnTRXActivation      bool   `json:"use_burn_trx_activation" yaml:"use_burn_trx_activation" default:"true"`
}

func (s TronBlockchain) Validate() error {
	if !s.Enabled {
		return nil
	}

	if s.Node.GrpcAddress == "" {
		return fmt.Errorf("tron: node gRPC address must not be empty")
	}

	return nil
}

func (s TronBlockchain) ConvertToSDKConfig(identity constants.ProcessingIdentity) tron.Config {
	return tron.Config{
		NodeAddr:                  s.Node.GrpcAddress,
		UseTLS:                    s.Node.UseTLS,
		ActivationContractAddress: s.ActivationContractAddress,
		UseBurnTRXActivation:      s.UseBurnTRXActivation,
		GRPCOptions: []grpc.DialOption{
			grpc.WithUnaryInterceptor(tron.PrepareUnaryInterceptor(
				constants.ProcessingIDParamName.String(),
				identity.ID,
				constants.ProcessingVersionParamName.String(),
				identity.Version,
			)),
			grpc.WithStreamInterceptor(tron.PrepareStreamInterceptor(
				constants.ProcessingIDParamName.String(),
				identity.ID,
				constants.ProcessingVersionParamName.String(),
				identity.Version,
			),
			),
		},
	}
}
