package tron

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type ChainParams struct {
	EnergyFee                           int64
	TransactionFee                      int64
	TotalEnergyCurrentLimit             int64
	FreeNetLimit                        int64
	CreateNewAccountFeeInSystemContract int64
	CreateAccountFee                    int64
}

// ChainParam get chain parameters
func (t *Tron) ChainParam(ctx context.Context, paramKey string) (*core.ChainParameters_ChainParameter, error) {
	data, err := t.node.Client.GetChainParameters(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, err
	}

	for _, item := range data.GetChainParameter() {
		if item.Key == paramKey {
			return item, nil
		}
	}

	return nil, fmt.Errorf("chain parameter not found")
}

func (t *Tron) ChainParams(ctx context.Context) (*ChainParams, error) {
	data, err := t.node.Client.GetChainParameters(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, err
	}

	res := &ChainParams{}
	for _, item := range data.GetChainParameter() {
		switch item.Key {
		case "getEnergyFee":
			res.EnergyFee = item.Value
		case "getTransactionFee":
			res.TransactionFee = item.Value
		case "getTotalEnergyCurrentLimit":
			res.TotalEnergyCurrentLimit = item.Value
		case "getFreeNetLimit":
			res.FreeNetLimit = item.Value
		case "getCreateAccountFee":
			res.CreateAccountFee = item.Value
		case "getCreateNewAccountFeeInSystemContract":
			res.CreateNewAccountFeeInSystemContract = item.Value
		}
	}

	return res, nil
}
