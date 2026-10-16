package fsmtron

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/rpccode"
)

// refundAmount returns the amount sent to the recipient of a refund. The hot wallet pays the fee by itself:
// for TRX it is subtracted from the amount, for TRC20 the hot wallet must hold enough TRX for burning.
func (s *FSM) refundAmount(ctx context.Context, fromAddress, toAddress string, amount decimal.Decimal) (decimal.Decimal, error) {
	assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get asset decimals: %w", err)
	}

	estimate, err := s.tron.EstimateTransferResources(ctx, fromAddress, toAddress, s.transfer.AssetIdentifier, amount, assetDecimals)
	if err != nil {
		return decimal.Zero, fmt.Errorf("estimate transfer resources: %w", err)
	}

	resources, err := s.tron.TotalAvailableResources(fromAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get total available resources: %w", err)
	}

	fee, err := s.tron.EstimateSelfFundedTransferFee(ctx, fromAddress, toAddress, s.transfer.AssetIdentifier, *estimate, *resources)
	if err != nil {
		return decimal.Zero, err
	}

	sentAmount := amount
	if s.transfer.AssetIdentifier == tron.TrxAssetIdentifier {
		sentAmount = amount.Sub(fee)
		if !sentAmount.IsPositive() {
			return decimal.Zero, fmt.Errorf("%w: refund amount %s does not cover fee %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeRefundInsufficientFee), amount, fee)
		}
	} else {
		trxBalance, err := s.getBalance(ctx, fromAddress, tron.TrxAssetIdentifier)
		if err != nil {
			return decimal.Zero, fmt.Errorf("get trx balance: %w", err)
		}

		if trxBalance.LessThan(fee) {
			return decimal.Zero, fmt.Errorf("%w: trx balance %s, fee %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeRefundInsufficientFee), trxBalance, fee)
		}
	}

	if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, map[string]any{
		"refund": map[string]any{
			stateKeyAmount: amount.String(),
			"fee":          fee.String(),
			"sent_amount":  sentAmount.String(),
		},
	}); err != nil {
		return decimal.Zero, fmt.Errorf("set state data: %w", err)
	}

	return sentAmount, nil
}
