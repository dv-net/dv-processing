package fsmtron

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/jackc/pgx/v5"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
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

	if _, err := getTransferKind(s.transfer.Kind.String); err != nil {
		return err
	}

	// check cold or processing wallet
	if s.transfer.WalletFromType == constants.WalletTypeHot {
		checkResult, err := s.bs.Wallets().CheckWallet(ctx, wconstants.BlockchainTypeTron, s.transfer.GetToAddress())
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

	return s.setTransferStatus(ctx, constants.TransferStatusProcessing)
}

// checkActivateWallet
func (s *FSM) checkActivateWallet(ctx context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	// if trx transfer, skip activation and go to the check transfer kind step
	if s.transfer.AssetIdentifier == tron.TrxAssetIdentifier {
		w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
		return nil
	}

	var isActivated bool
	switch s.transfer.WalletFromType {
	case constants.WalletTypeHot:
		// get hot hotWallet
		hotWallet, err := s.bs.Wallets().Hot().Get(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, s.transfer.GetFromAddress())
		if err != nil {
			return fmt.Errorf("get hot wallet: %w", err)
		}
		// if wallet is activated, go to the check transfer kind step
		if hotWallet.IsActivated {
			w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
			return nil
		}
		// find trx transactions
		isActivated, err = s.tron.CheckIsWalletActivated(hotWallet.Address)
		if err != nil {
			return fmt.Errorf("check is wallet activated: %w", err)
		}
		// if there are transactions, activate the wallet and go to the check transfer kind step
		if isActivated {
			if err := s.bs.Wallets().Hot().ActivateWallet(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, hotWallet.Address); err != nil {
				return err
			}
			w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
			return nil
		}
	case constants.WalletTypeProcessing:
		// get processing wallet
		processingWallet, err := s.bs.Wallets().Processing().Get(ctx, wconstants.BlockchainTypeTron, s.transfer.FromAddresses[0])
		if err != nil {
			return fmt.Errorf("get processing wallet: %w", err)
		}
		// if wallet is activated, go to the check transfer kind step
		isActivated, err = s.tron.CheckIsWalletActivated(processingWallet.Address)
		if err != nil {
			return fmt.Errorf("check is processing wallet activated: %w", err)
		}
		if isActivated {
			w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
			return nil
		}
	case constants.WalletTypeCold:
		return fmt.Errorf("cold wallet activation is not supported in tron fsm")
	}

	w.State.SetNextStep(stepBeforeActivationCheckTransferKind)
	return nil
}

func (s *FSM) beforeActivationCheckTransferKind(_ context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	transferKind, err := getTransferKind(s.transfer.Kind.String)
	if err != nil {
		return err
	}

	switch transferKind {
	case constants.TronTransferKindBurnTRX:
		if s.transfer.WalletFromType != constants.WalletTypeHot {
			return fmt.Errorf("activation for burn trx transfer is only supported for hot wallets")
		}
		if s.config.Blockchain.Tron.UseBurnTRXActivation {
			wf.State.SetNextStep(stepActivateWalletBurnTRX)
			return nil
		}
		wf.State.SetNextStep(stepActiveWalletResources)
		return nil
	case constants.TronTransferKindResources:
		if s.transfer.WalletFromType != constants.WalletTypeHot {
			return fmt.Errorf("activation for resources transfer is only supported for hot wallets")
		}
		if s.config.Blockchain.Tron.UseBurnTRXActivation {
			wf.State.SetNextStep(stepActivateWalletBurnTRX)
			return nil
		}
		wf.State.SetNextStep(stepActiveWalletResources)
		return nil
	case constants.TronTransferKindCloudDelegate:
		wf.State.SetNextStep(stepWaitingExternalActivateWalletConfirmations)
		return nil
	default:
		return fmt.Errorf("unsupported transfer kind: %s", s.transfer.Kind.String)
	}
}

func (s *FSM) activateWalletResources(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	processingWallet, err := s.bs.Wallets().Processing().GetByBlockchain(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}
	estimate, err := s.tron.EstimateExternalContractActivation(ctx, processingWallet.Address, s.transfer.GetFromAddress())
	if err != nil {
		return fmt.Errorf("estimate activation call: %w", err)
	}

	estimate.Energy = estimate.Energy.Mul(tron.ResourceCoefficient).Ceil()
	estimate.Bandwidth = estimate.Bandwidth.Mul(tron.ResourceCoefficient).Ceil()

	_, err = step.State.SetArgs(map[string]any{
		"activation_estimate": estimate,
	})
	if err != nil {
		return fmt.Errorf("set args: %w", err)
	}

	resourcesData, err := s.tron.TotalAvailableResources(processingWallet.Address)
	if err != nil {
		return fmt.Errorf("get total available resources: %w", err)
	}

	_, err = step.State.SetArgs(map[string]any{
		"processing_resources": struct {
			Energy    decimal.Decimal
			Bandwidth decimal.Decimal
		}{
			Energy:    resourcesData.Energy,
			Bandwidth: resourcesData.Bandwidth,
		},
	})
	if err != nil {
		return fmt.Errorf("set args: %w", err)
	}

	if resourcesData.Bandwidth.LessThan(estimate.Bandwidth) {
		return fmt.Errorf("not enough bandwidth for activation call")
	}
	if resourcesData.Energy.LessThan(estimate.Energy) {
		return fmt.Errorf("not enough energy for activation call")
	}

	tx, err := s.tron.CreateUnsignedActivationTransaction(ctx, processingWallet.Address, s.transfer.GetFromAddress(), false)
	if err != nil {
		return fmt.Errorf("create unsigned activation transaction: %w", err)
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	// sign transaction
	if err = s.tron.SignTransaction(tx.Transaction, wcreds.PrivateKey.ToECDSA()); err != nil {
		return fmt.Errorf("sign activation transaction: %w", err)
	}

	// broadcast with system tx persist
	return pgx.BeginTxFunc(ctx, s.st.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err = s.initPendingSystemTransaction(ctx, tx, dbTx); err != nil {
			return fmt.Errorf("store system transaction info: %w", err)
		}

		if _, err := s.tron.Node().Broadcast(tx.GetTransaction()); err != nil {
			return fmt.Errorf("broadcast transaction: %w", err)
		}

		return nil
	})
}

func (s *FSM) waitingResourcesActivateWalletConfirmations(ctx context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeAccountActivation)
	if err != nil {
		return fmt.Errorf("find transaction: %w", err)
	}
	if len(txs) != 1 {
		return fmt.Errorf("expected exectly 1 transaction, got %d; [transfer_id: %s]", len(txs), s.transfer.ID)
	}

	if err = s.checkTransactionConfirmations(ctx, txs[0], systemMinConfirmationsCount); err != nil {
		return err
	}

	if err = s.bs.Wallets().Hot().ActivateWallet(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, s.transfer.GetFromAddress()); err != nil {
		return err
	}

	w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
	return nil
}

func (s *FSM) waitingExternalActivateWalletConfirmations(ctx context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	stateData, err := s.st.Transfers().GetStateData(ctx, s.transfer.ID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	activationOrderID, err := util.GetByPath[string](stateData, ActivationOrderID)
	if err != nil && !errors.Is(err, util.ErrGetByPathNotFound) {
		return fmt.Errorf("get by path: %w", err)
	}

	if err := s.checkExternalOrderStatus(ctx, activationOrderID); err != nil {
		return err
	}

	if err := s.bs.Wallets().Hot().ActivateWallet(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, s.transfer.GetFromAddress()); err != nil {
		return err
	}

	w.State.SetNextStep(stepBeforeSendingCheckTransferKind)
	return nil
}

func (s *FSM) activateWalletBurnTRX(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	// get processing wallet by owner id
	processingWallet, err := s.bs.Wallets().Processing().GetByBlockchain(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}

	// check processing wallet trx balance
	balance, err := s.getBalance(ctx, processingWallet.Address, tron.TrxAssetIdentifier)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	accountActivationFee, err := s.tron.EstimateActivationFee(ctx, processingWallet.Address, s.transfer.GetFromAddress())
	if err != nil {
		return fmt.Errorf("estimate system activation transaction: %w", err)
	}

	if accountActivationFee.Trx.IsZero() {
		step.State.SetNextStep(stepBeforeSendingCheckTransferKind)
		return nil
	}

	if balance.LessThanOrEqual(accountActivationFee.Trx) {
		return fmt.Errorf("processing wallet balance is less than %s trx", accountActivationFee.Trx.String())
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	_, err = s.systemActivation(ctx, wcreds, s.transfer.GetFromAddress())
	if err != nil {
		return fmt.Errorf("system activation: %w", err)
	}

	return nil
}

// waitingActivateWalletConfirmations
func (s *FSM) waitingActivateWalletConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeAccountActivation)
	if err != nil {
		return fmt.Errorf("find activation transaction: %w", err)
	}
	if len(txs) != 1 {
		return fmt.Errorf("expected 1 activation transaction per tranfser, got %d", len(txs))
	}

	// check confirmations
	if err = s.checkTransactionConfirmations(ctx, txs[0], systemMinConfirmationsCount); err != nil {
		return err
	}

	return s.bs.Wallets().Hot().ActivateWallet(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, s.transfer.GetFromAddress())
}

// beforeSendingCheckTransferKind checks the transfer kind.
func (s *FSM) beforeSendingCheckTransferKind(_ context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	transferKind, err := getTransferKind(s.transfer.Kind.String)
	if err != nil {
		return err
	}

	switch transferKind {
	case constants.TronTransferKindBurnTRX:
		wf.State.SetNextStep(stepSendTRXForBurn)
		return nil
	case constants.TronTransferKindResources:
		wf.State.SetNextStep(stepDelegateResources)
		return nil
	case constants.TronTransferKindCloudDelegate:
		wf.State.SetNextStep(stepWaitingExternalDelegateConfirmations)
		return nil
	default:
		return fmt.Errorf("unsupported transfer kind: %s", s.transfer.Kind.String)
	}
}

// sendTRXForBurning
func (s *FSM) sendTRXForBurn(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if s.transfer.WalletFromType == constants.WalletTypeProcessing {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	if s.transfer.WalletFromType != constants.WalletTypeHot {
		return fmt.Errorf("unsupported wallet type: %s", s.transfer.WalletFromType)
	}

	// skip delegate trx for trx transfers
	if s.transfer.AssetIdentifier == tron.TrxAssetIdentifier {
		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)
		return nil
	}

	// get processing wallet by owner id
	processingWallet, err := s.bs.Wallets().Processing().GetByBlockchain(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}

	assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get asset decimals: %w", err)
	}

	amount := s.transfer.Amount.Decimal
	if s.transfer.WholeAmount {
		hotWalletBalance, err := s.getBalance(ctx, s.transfer.GetFromAddress(), s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		amount = hotWalletBalance
	}

	// estimate transfer resources
	estimate, err := s.tron.EstimateTransferResources(ctx, s.transfer.GetFromAddress(), s.transfer.GetToAddress(), s.transfer.AssetIdentifier, amount, assetDecimals)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	// check processing wallet trx balance
	balance, err := s.getBalance(ctx, processingWallet.Address, tron.TrxAssetIdentifier)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	sentTrxAmount := estimate.Trx

	chainParams, err := s.tron.ChainParams(ctx)
	if err != nil {
		return fmt.Errorf("gte chain params: %w", err)
	}

	resourcesData, err := s.tron.TotalAvailableResources(s.transfer.GetFromAddress())
	if err != nil {
		return fmt.Errorf("get total available resources: %w", err)
	}

	if resourcesData.Bandwidth.GreaterThanOrEqual(estimate.Bandwidth) {
		sentTrxAmount = sentTrxAmount.Sub(tron.NewBandwidth(estimate.Bandwidth).ToTRX(chainParams.TransactionFee).ToDecimal())
	}

	if balance.LessThan(sentTrxAmount) {
		return fmt.Errorf("not enough trx on processing wallet: %s", sentTrxAmount.String())
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	_, err = s.sendTRX(ctx, wcreds, s.transfer.GetFromAddress(), sentTrxAmount)
	if err != nil {
		return fmt.Errorf("send trx: %w", err)
	}

	return nil
}

// waitingSendBurningTrxConfirmations
func (s *FSM) waitingSendBurnTrxConfirmations(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeSendBurnBaseAsset)
	if err != nil {
		return fmt.Errorf("find burn txs: %w", err)
	}
	if len(txs) != 1 {
		return fmt.Errorf("expected exectly one tx, got: %d", len(txs))
	}

	// check confirmations
	if err = s.checkTransactionConfirmations(ctx, txs[0], systemMinConfirmationsCount); err != nil {
		return err
	}

	wf.State.SetNextStage(stageSending).SetNextStep(stepSending)

	return nil
}

// delegateResources or send trx for burning
func (s *FSM) delegateResources(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) (err error) {
	// get processing wallet by owner id
	processingWallet, err := s.bs.Wallets().Processing().GetByBlockchain(ctx, s.transfer.OwnerID, wconstants.BlockchainTypeTron)
	if err != nil {
		return fmt.Errorf("get processing wallet: %w", err)
	}

	assetDecimals, err := s.getAssetDecimals(ctx, s.transfer.AssetIdentifier)
	if err != nil {
		return fmt.Errorf("get asset decimals: %w", err)
	}

	amount := s.transfer.Amount.Decimal
	if s.transfer.WholeAmount {
		hotWalletBalance, err := s.getBalance(ctx, s.transfer.GetFromAddress(), s.transfer.AssetIdentifier)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		amount = hotWalletBalance
	}

	// estimate transfer resources
	estimate, err := s.tron.EstimateTransferResources(ctx,
		s.transfer.GetFromAddress(),
		s.transfer.GetToAddress(),
		s.transfer.AssetIdentifier,
		amount,
		assetDecimals,
	)
	if err != nil {
		return fmt.Errorf("estimate transfer resources: %w", err)
	}

	// get available resources on processing wallet
	processingResources, err := s.tron.AvailableForDelegateResources(ctx, processingWallet.Address)
	if err != nil {
		return fmt.Errorf("get processing wallet available resources: %w", err)
	}

	if s.transfer.WalletFromType == constants.WalletTypeProcessing {
		if processingResources.Bandwidth.LessThan(estimate.Bandwidth) {
			return fmt.Errorf("not enough bandwidth for transfer on processing wallet. required: %s, available: %s", estimate.Bandwidth, processingResources.Bandwidth)
		}

		if processingResources.Energy.LessThan(estimate.Energy) {
			return fmt.Errorf("not enough energy for transfer on processing wallet. required: %s, available: %s", estimate.Energy, processingResources.Energy)
		}

		wf.State.SetNextStage(stageSending).SetNextStep(stepSending)

		return nil
	}

	// get available resources on hot wallet
	hotWalletResources, err := s.tron.TotalAvailableResources(s.transfer.GetFromAddress())
	if err != nil {
		return fmt.Errorf("get hot wallet available resources: %w", err)
	}

	res, err := s.tron.EstimateTransferWithDelegateResources(ctx, tron.EstimateTransferWithDelegateResourcesRequest{
		ProcessingAddress:   processingWallet.Address,
		HotWalletAddress:    s.transfer.GetFromAddress(),
		ProcessingResources: *processingResources,
		HotResources:        *hotWalletResources,
		Estimate:            *estimate,
	})
	if err != nil {
		return fmt.Errorf("estimate transfer with delegate resources: %w", err)
	}

	if res.NeedBandwidthFromProcessingWallet.IsPositive() {
		if processingResources.Bandwidth.LessThan(res.NeedBandwidthFromProcessingWallet) {
			return fmt.Errorf("not enough bandwidth for transfer on processing wallet. required: %s, available: %s", res.NeedBandwidthFromProcessingWallet, processingResources.Bandwidth)
		}
	}

	// get wallet creds
	wcreds, err := s.getWalletCreds(ctx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("get wallet creds: %w", err)
	}

	stateData := xsync.NewMap[string, any]()
	stateData.Store("proecssing_wallet_available_resources", processingResources)
	stateData.Store("hot_wallet_available_resources", hotWalletResources)
	stateData.Store("estimated_resources_fsm_stage_before", estimate)

	defer func() {
		if stateData.Size() == 0 {
			return
		}

		stateDataMap := make(map[string]any, stateData.Size())
		stateData.Range(func(k string, v any) bool {
			stateDataMap[k] = v
			return true
		})

		if err := s.bs.Transfers().SetStateData(ctx, s.transfer.ID, stateDataMap); err != nil {
			s.logger.Errorf("set state data: %v", err)
		}
	}()

	eg, egCtx := errgroup.WithContext(ctx)
	for _, resourceType := range resourcesToDelegate {
		eg.Go(func() error {
			var amount decimal.Decimal
			switch resourceType {
			case core.ResourceCode_ENERGY:
				if !res.NeedToDelegate.Energy.IsPositive() {
					return nil
				}

				amount = res.NeedToDelegate.Energy
			case core.ResourceCode_BANDWIDTH:
				if !res.NeedToDelegate.Bandwidth.IsPositive() {
					return nil
				}

				amount = res.NeedToDelegate.Bandwidth

			default:
				return nil
			}

			tx, delegateData, err := s.delegateResource(egCtx, wcreds, s.transfer.GetFromAddress(), amount, resourceType)
			if err != nil {
				return fmt.Errorf("delegate %s resource: %w", resourceType.String(), err)
			}

			if delegateData == nil {
				return fmt.Errorf("delegate data is nil")
			}

			stateData.Store(getDelegateResourceKey(resourceType), delegateData)

			// set tx hash to the step args
			_, err = step.State.SetArg(strings.ToLower(fmt.Sprintf("delegate_%s_tx_hash", resourceType.String())), common.Bytes2Hex(tx.Txid))
			if err != nil {
				return fmt.Errorf("set arg: %w", err)
			}

			_, err = step.State.SetArg(DelegateFromAddress, processingWallet.Address)
			if err != nil {
				return fmt.Errorf("set arg: %w", err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return newErrorFailedTransfer(err, s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}

	return nil
}

// waitingDelegateConfirmations
func (s *FSM) waitingDelegateConfirmations(ctx context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	delegationTxs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeDelegateResources)
	if err != nil {
		return fmt.Errorf("find delegate transactions: %w", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, tx := range delegationTxs {
		eg.Go(func() error {
			// check confirmations
			return s.checkTransactionConfirmations(egCtx, tx, systemMinConfirmationsCount)
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	w.State.SetNextStage(stageSending).SetNextStep(stepSending)

	return nil
}

func (s *FSM) waitingExternalDelegateResources(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	orders := utils.NewSlice[string]()

	stateData, err := s.bs.Transfers().GetStateData(ctx, s.transfer.ID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	for _, resourceType := range resourcesToDelegate {
		argsKey := EnergyOrderID
		if resourceType == core.ResourceCode_BANDWIDTH {
			argsKey = BandwidthOrderID
		}
		delegateOrderID, err := util.GetByPath[string](stateData, argsKey)
		if err != nil && !errors.Is(err, util.ErrGetByPathNotFound) {
			return fmt.Errorf("get by path: %w", err)
		}
		if delegateOrderID != "" {
			orders.Add(delegateOrderID)
		}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, orderID := range orders.GetAll() {
		eg.Go(func() error {
			if err := s.checkExternalOrderStatus(egCtx, orderID); err != nil {
				return err
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
