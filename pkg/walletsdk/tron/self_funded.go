package tron

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// EstimateSelfFundedTransferFee returns TRX burned by the sender when it pays the transfer by itself,
// without delegated resources or activation from another wallet.
//
// For TRX it includes the activation of the recipient and the bandwidth if the sender lacks it.
// For TRC20 the whole estimated TRX is required if the sender lacks energy or bandwidth.
func (t *Tron) EstimateSelfFundedTransferFee(
	ctx context.Context,
	fromAddress, toAddress, assetIdentifier string,
	estimate EstimateTransferResourcesResult,
	resources Resources,
) (decimal.Decimal, error) {
	if assetIdentifier == TrxAssetIdentifier {
		activation, err := t.EstimateActivationFee(ctx, fromAddress, toAddress)
		if err != nil {
			return decimal.Zero, fmt.Errorf("estimate recipient activation fee: %w", err)
		}

		fee := activation.Trx
		if resources.Bandwidth.LessThan(estimate.Bandwidth) {
			fee = fee.Add(estimate.Trx)
		}

		return fee, nil
	}

	if resources.Energy.LessThan(estimate.Energy) || resources.Bandwidth.LessThan(estimate.Bandwidth) {
		return estimate.Trx, nil
	}

	return decimal.Zero, nil
}
