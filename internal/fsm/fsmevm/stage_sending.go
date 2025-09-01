package fsmevm

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/ethereum/go-ethereum/core/types"
)

// sending
func (s *FSM) sending(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	var err error

	fromAddress := s.transfer.GetFromAddress()
	toAddress := s.transfer.GetToAddress()

	// by default use amount from transfer request
	amount := s.transfer.Amount.Decimal

	// if whole amount is true, get wallet balance and use it as amount
	if s.transfer.WholeAmount {
		balance, err := s.getBalance(ctx, fromAddress, s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		amount = balance
	}

	// get asset decimals
	assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get asset decimals: %w", err)
	}

	currentEstimateResult, err := s.evm.EstimateTransfer(ctx, fromAddress, toAddress, s.transfer.AssetIdentifier, amount, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer: %w", err)
	}

	estimateResult := new(evm.EstimateTransferResult)
	*estimateResult = *currentEstimateResult

	if s.transfer.AssetIdentifier != s.evm.Blockchain().GetAssetIdentifier() { //nolint:nestif
		stateData, err := s.bs.Transfers().GetStateData(ctx, s.transfer.ID)
		if err != nil {
			return fmt.Errorf("get state data: %w", err)
		}

		res, err := util.GetByPath[evm.EstimateTransferResult](stateData, fmt.Sprintf("%s.send_result.estimated_data", SendBurnBaseAsset))
		if err != nil && !errors.Is(err, util.ErrGetByPathNotFound) {
			return fmt.Errorf("get by path: %w", err)
		}

		if err == nil && res.TotalFeeAmount.LessThan(estimateResult.TotalFeeAmount) {
			*estimateResult = res

			if currentEstimateResult.Estimate.SuggestGasPrice.GreaterThan(estimateResult.Estimate.MaxFeePerGas) {
				return fmt.Errorf("base fee is greater than max fee per gas, base fee %s / max fee per gas %s", estimateResult.Estimate.SuggestGasPrice, estimateResult.Estimate.MaxFeePerGas)
			}

			if currentEstimateResult.EstimateGasAmount.GreaterThan(estimateResult.EstimateGasAmount) {
				return fmt.Errorf("estimated gas amount is greater than previous, estimated gas amount %s / previous %s", currentEstimateResult.EstimateGasAmount, estimateResult.EstimateGasAmount)
			}
		}
	}

	if s.transfer.WholeAmount && s.transfer.AssetIdentifier == s.evm.Blockchain().GetAssetIdentifier() {
		amount = amount.Sub(evm.NewUnit(estimateResult.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal())
	}

	if !amount.IsPositive() {
		return fmt.Errorf("transfer amount is less than or equal to zero, amount %s / fee %s", amount, evm.NewUnit(estimateResult.TotalFeeAmount, evm.EtherUnitWei).Decimal())
	}

	// get sequence for wallet
	sequence, err := s.bs.Wallets().GetSequenceByWalletType(ctx, s.transfer.WalletFromType, s.transfer.OwnerID, s.evm.Blockchain(), fromAddress)
	if err != nil {
		return fmt.Errorf("get sequence by wallet type: %w", err)
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	var stateData map[string]any
	var newTx *types.Transaction

	if s.transfer.AssetIdentifier == s.evm.Blockchain().GetAssetIdentifier() {
		newTx, stateData, err = s.sendBaseAsset(ctx, wcreds, toAddress, amount, estimateResult)
		if err != nil {
			return fmt.Errorf("send %s: %w", s.evm.Blockchain().GetAssetIdentifier(), err)
		}
	} else {
		newTx, stateData, err = s.sendERC20(ctx, wcreds, s.transfer.AssetIdentifier, toAddress, amount, assetDecimals, estimateResult)
		if err != nil {
			return fmt.Errorf("send erc20: %w", err)
		}
	}

	if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, map[string]any{
		"send_asset": stateData,
	}); err != nil {
		return fmt.Errorf("set state data: %w", err)
	}

	// update transfer and set tx hash
	s.transfer, err = s.bs.Transfers().SetTxHash(ctx, s.transfer.ID, newTx.Hash().Hex())
	if err != nil {
		return fmt.Errorf("set tx hash: %w", err)
	}

	return nil
}
