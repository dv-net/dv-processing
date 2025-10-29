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
	estimate, err := s.EstimateFee(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate fee: %w", err)
	}

	var gasAmount decimal.Decimal
	if assetIdentifier == s.config.Blockchain.GetAssetIdentifier() {
		gasAmount, err = s.estimateNativeAssetGas(ctx, fromAddress, toAddress, amount)
	} else {
		gasAmount, err = s.estimateTokenGas(ctx, fromAddress, toAddress, assetIdentifier, amount, decimals)
	}
	if err != nil {
		return nil, err
	}

	gasTipCap := estimate.SuggestGasTipCap
	totalGasPrice := estimate.MaxFeePerGas
	totalFeeAmount := totalGasPrice.Mul(gasAmount)

	return &EstimateTransferResult{
		TotalFeeAmount:    totalFeeAmount,
		TotalGasPrice:     totalGasPrice,
		GasTipCap:         gasTipCap,
		EstimateGasAmount: gasAmount,
		Estimate:          *estimate,
	}, nil
}

// estimateNativeAssetGas estimates gas for native blockchain asset transfers
func (s *EVM) estimateNativeAssetGas(ctx context.Context, fromAddress, toAddress string, amount decimal.Decimal) (decimal.Decimal, error) {
	estimatedGas, err := s.node.EstimateGas(ctx, ethereum.CallMsg{
		From:  common.HexToAddress(fromAddress),
		To:    utils.Pointer(common.HexToAddress(toAddress)),
		Value: NewUnit(amount, EtherUnitEther).Value(EtherUnitWei).BigInt(),
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to estimate gas for eth: %w", err)
	}

	// Use the actual gas limit that will be used in transaction
	gasLimit := GasLimitByBlockchain(s.config.Blockchain)

	if estimatedGas > gasLimit {
		return decimal.NewFromUint64(estimatedGas), nil
	}
	return decimal.NewFromUint64(gasLimit), nil
}

// estimateTokenGas estimates gas for token (ERC-20) transfers
func (s *EVM) estimateTokenGas(ctx context.Context, fromAddress, toAddress, assetIdentifier string, amount decimal.Decimal, decimals int64) (decimal.Decimal, error) {
	// Convert amount to token's smallest unit
	amount = amount.Mul(decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals))))

	data, err := s.abi.Pack("transfer", common.HexToAddress(toAddress), amount.BigInt())
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to pack transfer data: %w", err)
	}

	estimatedGas, err := s.node.EstimateGas(ctx, ethereum.CallMsg{
		From: common.HexToAddress(fromAddress),
		To:   utils.Pointer(common.HexToAddress(assetIdentifier)),
		Data: data,
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to estimate gas for contract: %w", err)
	}

	return decimal.NewFromUint64(estimatedGas), nil
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

	// Apply blockchain-specific minimum gas tip caps
	minGasTipCap := getMinGasTipCapByBlockchain(s.config.Blockchain)
	if fee.SuggestGasTipCap.LessThan(minGasTipCap) {
		fee.SuggestGasTipCap = minGasTipCap
	}

	fee.MaxFeePerGas = GetBaseFeeMultiplier(fee.SuggestGasPrice)

	if fee.MaxFeePerGas.LessThan(fee.SuggestGasPrice.Add(fee.SuggestGasTipCap)) {
		fee.MaxFeePerGas = fee.SuggestGasPrice.Add(fee.SuggestGasTipCap)
	}

	return fee, nil
}

// getMinGasTipCapByBlockchain returns the minimum gas tip cap for a given blockchain in Wei.
func getMinGasTipCapByBlockchain(blockchain wconstants.BlockchainType) decimal.Decimal {
	switch blockchain {
	case wconstants.BlockchainTypeBinanceSmartChain:
		return UnitMap[EtherUnitGWei].Div(decimal.NewFromInt(10))
	case wconstants.BlockchainTypeEthereum:
		return UnitMap[EtherUnitGWei]
	case wconstants.BlockchainTypePolygon:
		return decimal.NewFromInt(30).Mul(UnitMap[EtherUnitGWei])
	case wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return UnitMap[EtherUnitGWei].Div(decimal.NewFromInt(100))
	default:
		// Default: 1 Gwei
		return UnitMap[EtherUnitGWei]
	}
}

// GasLimitByBlockchain returns the gas limit by blockchain for base asset transfers.
func GasLimitByBlockchain(blockchain wconstants.BlockchainType) uint64 {
	switch blockchain {
	case wconstants.BlockchainTypeArbitrum:
		return 38000
	default:
		return 21000 // Default gas limit for unknown blockchains
	}
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
