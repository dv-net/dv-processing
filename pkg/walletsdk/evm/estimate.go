package evm

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type EstimateTransferResult struct {
	Estimate          EstimateFeeResult `json:"estimate"`
	EstimateGasAmount decimal.Decimal   `json:"estimate_gas_amount"`
	TotalFeeAmount    decimal.Decimal   `json:"total_fee_amount"`
	TotalGasPrice     decimal.Decimal   `json:"total_gas_price"`
	GasTipCap         decimal.Decimal   `json:"gas_tip_cap"`
}

// EstimateTransfer estimates the transfer fee. Amount is in Eth
func (s *EVM) EstimateTransfer(ctx context.Context, fromAddress, toAddress, assetIdentifier string, amount decimal.Decimal, decimals int64) (*EstimateTransferResult, error) {
	var gasAmount decimal.Decimal
	var gasTipCap decimal.Decimal
	var totalFeeAmount decimal.Decimal
	var totalGasPrice decimal.Decimal

	estimate, err := s.EstimateFee(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate fee: %w", err)
	}

	if assetIdentifier == s.config.Blockchain.GetAssetIdentifier() { //nolint:nestif
		estimatedGas, err := s.node.EstimateGas(ctx, ethereum.CallMsg{
			From:  common.HexToAddress(fromAddress),
			To:    utils.Pointer(common.HexToAddress(toAddress)),
			Value: NewUnit(amount, EtherUnitEther).Value(EtherUnitWei).BigInt(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas for eth: %w", err)
		}

		gasAmount = decimal.NewFromUint64(estimatedGas)
		gasTipCap = estimate.SuggestGasTipCap

		if gasTipCap.GreaterThan(estimate.MaxFeePerGas) {
			gasTipCap = estimate.MaxFeePerGas
		}

		totalGasPrice = estimate.MaxFeePerGas.Add(gasTipCap)
		totalFeeAmount = totalGasPrice.Mul(gasAmount)
	} else {
		amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals))))

		data, err := s.abi.Pack("transfer", common.HexToAddress(toAddress), amount.BigInt())
		if err != nil {
			return nil, fmt.Errorf("failed to pack transfer data: %w", err)
		}

		estimatedGas, err := s.node.EstimateGas(ctx, ethereum.CallMsg{
			From: common.HexToAddress(fromAddress),
			To:   utils.Pointer(common.HexToAddress(assetIdentifier)),
			Data: data,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas for contract: %w", err)
		}

		gasAmount = decimal.NewFromUint64(estimatedGas)
		gasTipCap = estimate.SuggestGasTipCap

		if gasTipCap.GreaterThan(estimate.MaxFeePerGas) {
			gasTipCap = estimate.MaxFeePerGas
		}

		totalGasPrice = estimate.MaxFeePerGas.Add(gasTipCap)
		totalFeeAmount = totalGasPrice.Mul(gasAmount)
	}

	return &EstimateTransferResult{
		TotalFeeAmount:    totalFeeAmount,
		TotalGasPrice:     totalGasPrice,
		GasTipCap:         gasTipCap,
		EstimateGasAmount: gasAmount,
		Estimate:          *estimate,
	}, nil
}

// EstimateFeeResult represents the result of the fee estimation
//
// All values are in Wei
type EstimateFeeResult struct {
	// MaxFeePerGas is the maximum fee per gas in Wei
	MaxFeePerGas decimal.Decimal `json:"max_fee_per_gas"`
	// SuggestGasTipCap is the suggested gas tip cap in Wei
	SuggestGasTipCap decimal.Decimal `json:"suggest_gas_tip_cap"`
	// SuggestGasPrice is the suggested gas price in Wei
	SuggestGasPrice decimal.Decimal `json:"suggest_gas_price"`
}

func (s *EVM) EstimateFee(ctx context.Context) (*EstimateFeeResult, error) {
	var (
		gasTipCap       decimal.Decimal
		suggestGasPrice decimal.Decimal
	)

	eg, egCtx := errgroup.WithContext(ctx)

	// get gas tip cap
	eg.Go(func() error {
		res, err := s.node.SuggestGasTipCap(egCtx)
		if err != nil {
			return err
		}
		gasTipCap = decimal.NewFromBigInt(res, 0)
		return nil
	})

	// get suggest gas price
	eg.Go(func() error {
		res, err := s.node.SuggestGasPrice(egCtx)
		if err != nil {
			return err
		}
		suggestGasPrice = decimal.NewFromBigInt(res, 0)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	fee := &EstimateFeeResult{
		SuggestGasTipCap: gasTipCap,
		SuggestGasPrice:  suggestGasPrice,
	}

	if s.config.Blockchain == wconstants.BlockchainTypeEthereum {
		fee.SuggestGasTipCap = UnitMap[EtherUnitGWei]
	}

	if fee.SuggestGasTipCap.LessThan(UnitMap[EtherUnitGWei]) {
		fee.SuggestGasTipCap = UnitMap[EtherUnitGWei]
	}

	fee.MaxFeePerGas = GetBaseFeeMultiplier(fee.SuggestGasPrice)

	return fee, nil
}

func GetBaseFeeMultiplier(baseFeeWei decimal.Decimal) decimal.Decimal {
	items := []struct {
		threshold  int64
		multiplier decimal.Decimal
	}{
		{
			threshold:  200,
			multiplier: decimal.NewFromFloat(1.14),
		},
		{
			threshold:  100,
			multiplier: decimal.NewFromFloat(1.17),
		},
		{
			threshold:  40,
			multiplier: decimal.NewFromFloat(1.18),
		},
		{
			threshold:  20,
			multiplier: decimal.NewFromFloat(1.19),
		},
		{
			threshold:  10,
			multiplier: decimal.NewFromFloat(1.192),
		},
		{
			threshold:  9,
			multiplier: decimal.NewFromFloat(1.195),
		},
		{
			threshold:  8,
			multiplier: decimal.NewFromFloat(1.20),
		},
		{
			threshold:  7,
			multiplier: decimal.NewFromFloat(1.215),
		},
		{
			threshold:  6,
			multiplier: decimal.NewFromFloat(1.22),
		},
		{
			threshold:  5,
			multiplier: decimal.NewFromFloat(1.24),
		},
		{
			threshold:  4,
			multiplier: decimal.NewFromFloat(1.26),
		},
	}

	baseFeeGWei := NewUnit(baseFeeWei, EtherUnitWei).Value(EtherUnitGWei).Decimal()

	for _, item := range items {
		if baseFeeGWei.GreaterThanOrEqual(decimal.NewFromInt(item.threshold)) {
			return item.multiplier.Mul(baseFeeWei)
		}
	}

	return decimal.NewFromFloat(1.30).Mul(baseFeeWei)
}
