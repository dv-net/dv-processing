package fsmevm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/dv-processing/rpccode"
)

// validateRequest
func (s *FSM) validateRequest(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if len(s.transfer.FromAddresses) != 1 {
		return fmt.Errorf("required one from address")
	}

	if len(s.transfer.ToAddresses) != 1 {
		return fmt.Errorf("required one to address")
	}

	if s.transfer.AssetIdentifier == "" {
		return fmt.Errorf("asset identifier is empty")
	}

	// check cold or processing wallet
	if s.transfer.WalletFromType == constants.WalletTypeHot {
		checkResult, err := s.bs.Wallets().CheckWallet(ctx, s.evm.Blockchain(), s.transfer.GetToAddress())
		if err != nil {
			return fmt.Errorf("check wallet: %w", err)
		}

		if !slices.Contains([]constants.WalletType{constants.WalletTypeCold, constants.WalletTypeProcessing}, checkResult.WalletType) {
			return fmt.Errorf("invalid wallet type %s", checkResult.WalletType)
		}

		if checkResult.OwnerID != s.transfer.OwnerID {
			return fmt.Errorf("invalid wallet owner %s", checkResult.OwnerID)
		}
	}

	if s.transfer.WalletFromType == constants.WalletTypeHot && //nolint:nestif
		(s.transfer.Fee.Valid || s.config.GetMaxGasFee() > 0) {
		// by default use amount from transfer request
		amount := s.transfer.Amount.Decimal

		// if whole amount is true, get wallet balance and use it as amount
		if s.transfer.WholeAmount {
			balance, err := s.getBalance(ctx, s.transfer.GetFromAddress(), s.transfer.AssetIdentifier)
			if err != nil {
				return fmt.Errorf("get balance: %w", err)
			}

			amount = balance
		}

		assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get asset decimals: %w", err)
		}

		estimateResult, err := s.evm.EstimateTransfer(ctx, s.transfer.GetFromAddress(), s.transfer.GetToAddress(), s.transfer.AssetIdentifier, amount, assetDecimals)
		if err != nil {
			return fmt.Errorf("estimate transfer resources: %w", err)
		}

		var feeMax decimal.Decimal
		if s.transfer.FeeMax.Valid {
			feeMax = s.transfer.FeeMax.Decimal
		} else {
			feeMax = decimal.NewFromFloat(s.config.GetMaxGasFee())
		}

		totalGasPriceGwei := evm.NewUnit(estimateResult.TotalGasPrice, evm.EtherUnitWei).Value(evm.EtherUnitGWei).Decimal()

		if totalGasPriceGwei.GreaterThan(feeMax) {
			return fmt.Errorf("%w: estimated gas price is exceeded: %s > %s", rpccode.GetErrorByCode(rpccode.RPCCodeMaxFeeExceeded), totalGasPriceGwei, feeMax)
		}
	}

	return s.setTransferStatus(ctx, constants.TransferStatusProcessing)
}

// sendBaseAssetForBurn
func (s *FSM) sendBaseAssetForBurn(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	// skip processing wallets
	if s.transfer.WalletFromType == constants.WalletTypeProcessing {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	if s.transfer.WalletFromType != constants.WalletTypeHot {
		return fmt.Errorf("unsupported wallet type: %s", s.transfer.WalletFromType)
	}

	// skip sending base asset for the same asset
	if s.transfer.AssetIdentifier == s.evm.Blockchain().GetAssetIdentifier() {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	// get processing wallet by owner id
	processingWallet, err := s.bs.Wallets().Processing().GetByBlockchain(ctx, s.transfer.OwnerID, s.evm.Blockchain())
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}

	// get processing wallet balance
	processingWalletBaseAssetBalance, err := s.getBalance(ctx, processingWallet.Address, s.evm.Blockchain().GetAssetIdentifier())
	if err != nil {
		return fmt.Errorf("get processing %s balance: %w", s.evm.Blockchain().GetAssetIdentifier(), err)
	}

	hotWalletBaseAssetAmount, err := s.getBalance(ctx, s.transfer.GetFromAddress(), s.evm.Blockchain().GetAssetIdentifier())
	if err != nil {
		return fmt.Errorf("get hot wallet %s balance: %w", s.evm.Blockchain().GetAssetIdentifier(), err)
	}

	hotWalletAssetAmount := s.transfer.Amount.Decimal
	if s.transfer.WholeAmount {
		hotWalletBalance, err := s.getBalance(ctx, s.transfer.GetFromAddress(), s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		hotWalletAssetAmount = hotWalletBalance
	}

	assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get asset decimals: %w", err)
	}

	// estimateResult transfer resources
	estimateResult, err := s.evm.EstimateTransfer(ctx, s.transfer.GetFromAddress(), s.transfer.GetToAddress(), s.transfer.AssetIdentifier, hotWalletAssetAmount, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	// in base asset
	transferFeeBaseAssetAmount := evm.NewUnit(estimateResult.TotalFeeAmount.Mul(decimal.NewFromFloat(evm.TransferFeeCoeff)), evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal()

	diffHotWalletBaseAssetAndFee := hotWalletBaseAssetAmount.Sub(transferFeeBaseAssetAmount)

	// if diff is negative, skip sending base asset to hot wallet
	if diffHotWalletBaseAssetAndFee.IsPositive() {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	needFeeBaseAssetAmount := transferFeeBaseAssetAmount.Sub(hotWalletBaseAssetAmount)
	if !needFeeBaseAssetAmount.IsPositive() {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	// if processing wallet balance is less than fee amount
	if processingWalletBaseAssetBalance.LessThan(needFeeBaseAssetAmount) {
		return fmt.Errorf("not enough %s on processing wallet: fee %s / balance %s", s.evm.Blockchain().GetAssetIdentifier(), needFeeBaseAssetAmount.String(), processingWalletBaseAssetBalance.String())
	}

	// estimateResult transfer resources
	estimateSendBaseAssetResult, err := s.evm.EstimateTransfer(ctx, processingWallet.Address, s.transfer.GetFromAddress(), s.evm.Blockchain().GetAssetIdentifier(), needFeeBaseAssetAmount, evm.EVMAssetDecimals)
	if err != nil {
		return fmt.Errorf("estimate %s transfer resources: %w", s.evm.Blockchain().GetAssetIdentifier(), err)
	}

	currentTransferFeeAmount := evm.NewUnit(estimateSendBaseAssetResult.TotalFeeAmount, evm.EtherUnitWei).Value(evm.EtherUnitEther).Decimal()

	// if processing wallet balance is less than fee amount
	if processingWalletBaseAssetBalance.Sub(currentTransferFeeAmount).LessThan(needFeeBaseAssetAmount) {
		return fmt.Errorf(
			"not enough %s on processing wallet: fee %s / transfer cost %s / balance %s",
			s.evm.Blockchain().GetAssetIdentifier(),
			needFeeBaseAssetAmount.String(),
			currentTransferFeeAmount.String(),
			processingWalletBaseAssetBalance.String(),
		)
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	_, sendStateData, err := s.sendBaseAsset(ctx, wcreds, s.transfer.GetFromAddress(), needFeeBaseAssetAmount, estimateResult)
	if err != nil {
		return fmt.Errorf("send %s: %w", s.evm.Blockchain().GetAssetIdentifier(), err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		defer wg.Done()

		fn := func() error {
			sendBurnBaseAssetStateData := map[string]any{
				"from":                processingWallet.Address,
				"to":                  s.transfer.GetFromAddress(),
				"transfer_fee_amount": transferFeeBaseAssetAmount.String(),
				"need_fee_amount":     needFeeBaseAssetAmount.String(),
				s.stringForBaseAsset("diff_hot_wallet_%s_and_fee"): diffHotWalletBaseAssetAndFee.String(),
				s.stringForBaseAsset("hot_wallet_%s_amount"):       hotWalletBaseAssetAmount.String(),
				"hot_wallet_asset_amount":                          hotWalletAssetAmount.String(),
				s.stringForBaseAsset("processing_wallet_%s"):       processingWalletBaseAssetBalance.String(),
				"current_transfer_fee_amount":                      currentTransferFeeAmount.String(),
				"send_result":                                      sendStateData,
			}

			if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, map[string]any{
				SendBurnBaseAsset: sendBurnBaseAssetStateData,
			}); err != nil {
				return fmt.Errorf("set state data: %w", err)
			}
			return nil
		}

		if err := fn(); err != nil {
			s.logger.Error(err)
		}
	}()

	wg.Wait()

	return nil
}

// waitingSendBurnBaseAssetConfirmations
func (s *FSM) waitingSendBurnBaseAssetConfirmations(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeSendBurnBaseAsset)
	if err != nil {
		return err
	}
	if len(txs) != 1 {
		return fmt.Errorf("expected exectly one tx, got: %d", len(txs))
	}

	// check tx in blockchain
	if err = s.ensureTxInBlockchain(ctx, txs[0].TxHash); err != nil {
		return err
	}

	wf.State.SetNextStage(stageSending).SetNextStep(stepSending)

	return nil
}
