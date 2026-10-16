package transfers

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
)

// processTronRefund checks that the hot wallet pays the refund by itself, without activation,
// delegated resources or TRX from the processing wallet:
//   - TRX: activation of the recipient and burned bandwidth are subtracted from the amount
//   - TRC20: own energy and bandwidth of the hot wallet or its own TRX for burning
func (s *Service) processTronRefund(ctx context.Context, req *CreateTransferRequest) error {
	fromAddress, toAddress := req.FromAddresses[0], req.ToAddresses[0]

	activated, err := s.blockchains.Tron.CheckIsWalletActivated(fromAddress)
	if err != nil {
		return fmt.Errorf("check wallet activation: %w", err)
	}

	if !activated {
		return fmt.Errorf("%w: hot wallet %s is not activated", rpccode.GetErrorByCode(rpccode.RPCCodeRefundInsufficientFee), fromAddress)
	}

	balance, err := s.eproxySvc.AddressBalance(ctx, fromAddress, req.AssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get hot wallet balance: %w", err)
	}

	if balance.LessThan(req.Amount.Decimal) {
		return fmt.Errorf("%w for refund. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeNotEnoughBalance), req.Amount.Decimal, balance)
	}

	assetDecimals, err := s.eproxySvc.AssetDecimals(ctx, wconstants.BlockchainTypeTron, req.AssetIdentifier)
	if err != nil {
		return err
	}

	estimate, err := s.blockchains.Tron.EstimateTransferResources(ctx, fromAddress, toAddress, req.AssetIdentifier, req.Amount.Decimal, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	resources, err := s.blockchains.Tron.TotalAvailableResources(fromAddress)
	if err != nil {
		return fmt.Errorf("get hot wallet available resources: %w", err)
	}

	fee, err := s.blockchains.Tron.EstimateSelfFundedTransferFee(ctx, fromAddress, toAddress, req.AssetIdentifier, *estimate, *resources)
	if err != nil {
		return err
	}

	req.stateData = map[string]any{
		"refund_estimated_resources": estimate,
		"refund_estimated_fee":       fee,
	}
	req.refundEstimatedFee = &fee

	if req.AssetIdentifier == tron.TrxAssetIdentifier {
		if req.Amount.Decimal.LessThanOrEqual(fee) {
			return fmt.Errorf("%w: refund amount %s does not cover fee %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeRefundInsufficientFee), req.Amount.Decimal, fee)
		}
		return nil
	}

	trxBalance, err := s.eproxySvc.AddressBalance(ctx, fromAddress, tron.TrxAssetIdentifier, req.Blockchain)
	if err != nil {
		return fmt.Errorf("get hot wallet trx balance: %w", err)
	}

	if trxBalance.LessThan(fee) {
		return fmt.Errorf("%w: trx balance %s, fee %s TRX", rpccode.GetErrorByCode(rpccode.RPCCodeRefundInsufficientFee), trxBalance, fee)
	}

	return nil
}
