package fsmtron

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/riverqueue/river"
	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

// waitingForTheFirstConfirmation
func (s *FSM) waitingForTheFirstConfirmation(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	// check confirmations
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeTransfer)
	if err != nil {
		return err
	}
	if len(txs) != 1 {
		return errors.New("expected 1 transaction")
	}

	if err = s.ensureTxInBlockchain(ctx, txs[0]); err != nil {
		return err
	}

	return s.setTransferStatus(ctx, constants.TransferStatusUnconfirmed)
}

// ensureTxInBlockchain - check if tx has received the first confirmation
func (s *FSM) ensureTxInBlockchain(ctx context.Context, transferTx *models.TransferTransaction) error {
	// get transaction info
	tx, err := s.bs.EProxy().GetTransactionInfo(ctx, wconstants.BlockchainTypeTron, transferTx.TxHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return workflow.NoConsoleError(river.JobSnooze(time.Second))
		}

		return fmt.Errorf("get transaction info: %w", err)
	}

	if tx.GetStatus() != "success" {
		if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusFailed); err != nil {
			return newErrorFailedTransfer(err, s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
		}

		return newErrorFailedTransfer(fmt.Errorf("transaction status is not success: %s", tx.GetStatus()), s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}

	// if transaction is not confirmed, snooze the job
	if tx.Confirmations < 1 {
		return workflow.NoConsoleError(river.JobSnooze(
			constants.ConfirmationsTimeoutWithRequired(wconstants.BlockchainTypeTron, 1, tx.Confirmations),
		))
	}

	if err = s.updateSystemTransactionStatus(ctx, tx, models.TransferTransactionsStatusUnconfirmed); err != nil {
		return newErrorFailedTransfer(err, s.wf.CurrentStep().Name, s.wf.CurrentStage().Name)
	}

	return nil
}

// waitingConfirmations
func (s *FSM) waitingConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	txs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeTransfer)
	if err != nil {
		return err
	}
	if len(txs) > 1 {
		return errors.New("expected only one main transaction")
	}

	if err = s.checkTransactionConfirmations(ctx, txs[0], constants.GetMinConfirmations(wconstants.BlockchainTypeTron)); err != nil {
		return err
	}

	return nil
}

// afterSendingCheckTransferKind checks the transfer kind.
func (s *FSM) afterSendingCheckTransferKind(_ context.Context, wf *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if s.transfer.WalletFromType == constants.WalletTypeProcessing {
		wf.State.SetNextStep(stepSendSuccessEvent)
		return nil
	}

	transferKind, err := getTransferKind(s.transfer.Kind.String)
	if err != nil {
		return err
	}

	switch transferKind {
	case constants.TronTransferKindBurnTRX:
		wf.State.SetNextStep(stepSendSuccessEvent)
		return nil
	case constants.TronTransferKindResources:
		wf.State.SetNextStep(stepReclaimResources)
		return nil
	case constants.TronTransferKindCloudDelegate:
		wf.State.SetNextStep(stepSendSuccessEvent)
		return nil
	default:
		return fmt.Errorf("unsupported transfer kind: %s", s.transfer.Kind.String)
	}
}

// reclaimResources
func (s *FSM) reclaimResources(ctx context.Context, wf *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	if s.transfer.WalletFromType != constants.WalletTypeHot {
		s.logger.Errorf("unsupported resources reclaim for wallet type %s", s.transfer.WalletFromType)
		wf.State.SetNextStep(stepSendSuccessEvent)
		return nil
	}

	processingWallet, err := workflow.GetArg[string](
		wf.GetSnapshot(),
		stepDelegateResources,
		DelegateFromAddress,
	)
	if err != nil {
		if errors.Is(err, workflow.ErrNotFound) {
			wf.State.SetNextStep(stepSendSuccessEvent)
			return nil
		}
		return err
	}

	_, frozenResources, err := s.tron.StakedResources(ctx, processingWallet)
	if err != nil {
		return fmt.Errorf("get staked resources: %w", err)
	}

	stateData := xsync.NewMap[string, any]()
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

	// there is no duplicate because of success and failed flow segregation
	for _, resourceType := range resourcesToDelegate { //nolint:dupl
		resourceKey := getDelegateResourceKey(resourceType)

		delegatedStateData, ok := s.transfer.StateData[resourceKey]
		if !ok {
			continue
		}

		eg.Go(func() error {
			delegatedData, err := utils.JSONToStruct[delegateStateData](delegatedStateData)
			if err != nil {
				return err
			}

			// get processing wallet by owner id
			processingWallet, err := s.bs.Wallets().Processing().GetByOwnerID(egCtx, s.transfer.OwnerID, wconstants.BlockchainTypeTron, delegatedData.FromAddress)
			if err != nil {
				return fmt.Errorf("get processing wallet: %w", err)
			}

			// get wallet creds
			wcreds, err := s.getWalletCreds(egCtx, s.transfer.OwnerID, uint32(processingWallet.Sequence)) //nolint:gosec
			if err != nil {
				return fmt.Errorf("get wallet creds: %w", err)
			}

			for _, frozenResource := range frozenResources {
				if frozenResource.DelegateTo != delegatedData.ToAddress {
					continue
				}

				if frozenResource.Type != resourceType {
					continue
				}

				if frozenResource.Amount == 0 {
					continue
				}

				var tx *api.TransactionExtention
				var reclaimData *reclaimStateData

				err := retry.New(
					retry.WithContext(ctx),
					retry.WithMaxAttempts(5),
					retry.WithDelay(time.Second*3),
				).Do(func() error {
					tx, reclaimData, err = s.reclaimResource(egCtx, wcreds, delegatedData.ToAddress, frozenResource.Amount, resourceType)
					if err != nil {
						if !strings.Contains(err.Error(), "account resource insufficient") {
							return fmt.Errorf("%w: %w", err, retry.ErrExit)
						}
					}
					return err
				})
				if err != nil {
					return fmt.Errorf("reclaim resource: %w", err)
				}

				if reclaimData == nil {
					return fmt.Errorf("reclaim data is nil")
				}

				stateData.Store(getReclaimResourceKey(resourceType), reclaimData)

				// set tx hash to the step args
				_, err = step.State.SetArg(
					strings.ToLower(fmt.Sprintf("reclaim_%s_tx_hash", resourceType.String())),
					common.Bytes2Hex(tx.Txid),
				)
				if err != nil {
					return fmt.Errorf("set arg: %w", err)
				}

				_, err = step.State.SetArg(ReclaimFromAddress, processingWallet.Address)
				if err != nil {
					return fmt.Errorf("set arg: %w", err)
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

// waitingReclaimResourcesConfirmations
func (s *FSM) waitingReclaimResourcesConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	reclaimTxs, err := s.st.TransferTransactions().FindTransactionByType(ctx, s.transfer.ID, models.TransferTransactionTypeReclaimResources)
	if err != nil {
		return fmt.Errorf("find reclaim txs: %w", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, reclaimTx := range reclaimTxs {
		eg.Go(func() error {
			return s.checkTransactionConfirmations(egCtx, reclaimTx, systemMinConfirmationsCount)
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	return nil
}

// sendSuccessEvent
func (s *FSM) sendSuccessEvent(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if err := s.setTransferStatus(ctx, constants.TransferStatusCompleted); err != nil {
		return err
	}

	return workflow.ErrExitWorkflow
}
