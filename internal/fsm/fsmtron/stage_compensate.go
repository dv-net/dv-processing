package fsmtron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/puzpuzpuz/xsync/v4"
	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func (s *FSM) determineCompensationFlow(_ context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	customError := s.wf.State.GetCustomError()
	if customError == nil {
		s.logger.Errorf("custom_error is nil, cannot proceed with compensation")
		w.State.SetNextStep(stepSendFailureEvent)
		return nil
	}

	failedTransferErr := &FailedTransferError{}
	if err := json.Unmarshal(customError, failedTransferErr); err != nil {
		s.logger.Errorf("failed to unmarshal custom_error: %v", err)
		w.State.SetNextStep(stepSendFailureEvent)
		return nil
	}

	// Compensation flow by transfer kind
	if s.transfer.Kind.String == string(constants.TronTransferKindResources) {
		w.State.SetNextStep(stepReclaimOnError)
		return nil
	}

	w.State.SetNextStep(stepSendFailureEvent)
	return nil
}

func (s *FSM) reclaimOnError(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, step *workflow.Step) error {
	processingWallet, err := workflow.GetArg[string](
		s.wf.GetSnapshot(),
		stepDelegateResources,
		DelegateFromAddress,
	)
	if err != nil {
		if errors.Is(err, workflow.ErrNotFound) {
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

				err = retry.New(
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

func (s *FSM) waitingReclaimOnErrorConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
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

// sendFailureEvent
func (s *FSM) sendFailureEvent(ctx context.Context, w *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	params, err := s.bs.Webhooks().EventTransferStatusCreateParams(ctx, webhooks.EventTransferStatusCreateParamsData{
		TransferID:   s.transfer.ID,
		OwnerID:      s.transfer.OwnerID,
		Status:       constants.TransferStatusFailed,
		ErrorMessage: s.wf.State.Error,
	})
	if err != nil {
		return fmt.Errorf("get event transfer status create params: %w", err)
	}

	w.State.SetFailed(true).SetError(err)
	w.SetSkipError(true)

	if err := s.bs.Transfers().SetWorkflowSnapshot(ctx, s.transfer.ID, w.GetSnapshot()); err != nil {
		return fmt.Errorf("set workflow snapshot: %w", err)
	}

	if err := s.bs.Webhooks().BatchCreate(ctx, []webhooks.BatchCreateParams{params}); err != nil {
		return fmt.Errorf("create failed event: %w", err)
	}

	if err := s.bs.Transfers().SetStatus(ctx, s.transfer.ID, constants.TransferStatusFailed); err != nil {
		return fmt.Errorf("set transfer status %s: %w", constants.TransferStatusFailed, err)
	}

	return nil
}
