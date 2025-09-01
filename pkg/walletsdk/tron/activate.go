package tron

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// EstimateActivationFee estimates the activation fee for a Tron address.
// It checks the available bandwidth and adds the activation fee accordingly.
// The fee is returned in TRX (1 TRX = 1_000_000 SUN).
// We assume that fromAddress is ALWAYS activated, being it processing address.
// Simple swap of arg to BlackHoleAddress on fakeTx creation will always return valid tx.
func (t *Tron) EstimateActivationFee(ctx context.Context, fromAddress, toAddress string) (*ActivationResources, error) {
	estimate := &ActivationResources{}

	isActivated, err := t.CheckIsWalletActivated(toAddress)
	if err != nil {
		return nil, fmt.Errorf("check wallet activation: %w", err)
	}

	if isActivated {
		return estimate, nil
	}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain params: %w", err)
	}

	// Add activation constant fee
	estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateNewAccountFeeInSystemContract))

	accountResources, err := t.AvailableForDelegateResources(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	// Estimate bandwidth required from staked bandwidth for activation transaction
	// BlackHoleAddress is safe to be used here on main/test networks.
	fakeTx, err := CreateFakeCreateAccountTransaction(fromAddress, toAddress)
	if err != nil {
		return nil, fmt.Errorf("fake create account tx: %w", err)
	}
	estimatedBandwidth, err := t.EstimateBandwidth(fakeTx)
	if err != nil {
		return nil, fmt.Errorf("estimate fake create account bandwidth: %w", err)
	}

	// Add 0.1 TRX when address does not have any staked bandwidth, nor does it have enough to activate account.
	// Or add actual bandwidth required if enough bandwidth
	if accountResources.Bandwidth.LessThan(estimatedBandwidth) {
		estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateAccountFee))
	} else {
		// We add coefficient to be safe.
		estimate.Bandwidth = estimate.Bandwidth.Add(estimatedBandwidth)
	}

	// Convert from SUN to TRX
	estimate.Trx = estimate.Trx.Div(decimal.NewFromInt(1e6))

	return estimate, nil
}
