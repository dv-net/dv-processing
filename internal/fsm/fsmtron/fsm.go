package fsmtron

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/mx/logger"
)

type FSM struct {
	logger   logger.Logger
	config   *config.Config
	wf       *workflow.Workflow
	transfer *models.Transfer
	st       store.IStore

	tron *tron.Tron

	// services
	bs baseservices.IBaseServices
}

func NewFSM( //nolint:funlen
	l logger.Logger,
	conf *config.Config,
	st store.IStore,
	bs baseservices.IBaseServices,
	transfer *models.Transfer,
) (*FSM, error) {
	fsm := &FSM{
		logger:   l,
		config:   conf,
		st:       st,
		bs:       bs,
		tron:     bs.Tron(),
		transfer: transfer,
	}

	// create a workflow
	fsm.wf = workflow.New(
		workflow.WithName("Tron FSM"),
		workflow.WithLogger(l),
		workflow.WithDebug(true),
		workflow.WithAfterAllStepsFn(func(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
		workflow.WithAfterFn(func(ctx context.Context, _ *workflow.Workflow) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
	).SetOnFailureFn(func(ctx context.Context, w *workflow.Workflow, err error) error {
		if err == nil {
			return nil
		}

		w.State.SetError(err)

		var failedTransferErr *FailedTransferError
		if errors.As(err, &failedTransferErr) {
			l.Infof(
				"failed transfer at stage %s, step %s: %v",
				failedTransferErr.FailedStage(),
				failedTransferErr.FailedStep(),
				failedTransferErr.Error(),
			)

			// Run compensation stage by specific failed transfer error
			serialized, serializeErr := failedTransferErr.MarshallJSON()
			if serializeErr != nil {
				return serializeErr
			}

			w.State.SetCustomError(serialized)
			w.State.SetNextStage(stageCompensateOnFail)
			w.State.SetNextStep(stepDetermineCompensationFlow)
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, w.GetSnapshot()); err != nil {
				l.Errorf("set workflow snapshot: %v", err)
			}
			return nil
		}

		if w.CurrentStage() != nil && w.CurrentStage().Name != "" && //nolint:nestif
			w.CurrentStep() != nil && w.CurrentStep().Name != "" {
			currentStage, currentStep := w.CurrentStage().Name, w.CurrentStep().Name

			// Run compensation stage by non-repeatable failed steps
			if steps, ok := exitOnFailedSteps[currentStage]; ok {
				for _, step := range steps {
					if step == currentStep {
						w.State.SetNextStage(stageCompensateOnFail)
						w.State.SetNextStep(stepDetermineCompensationFlow)
						if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, w.GetSnapshot()); err != nil {
							l.Errorf("set workflow snapshot: %v", err)
						}

						return nil
					}
				}
			}
		}

		return err
	}).SetBeforeAllStepsFn(func(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
		if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
			return fmt.Errorf("set workflow snapshot: %w", err)
		}

		return nil
	})

	// set stages and steps for the workflow
	fsm.wf.SetStages([]*workflow.Stage{
		{
			Name: stageBeforeSending,
			Steps: []*workflow.Step{
				{
					Name: stepValidateRequest,
					Func: fsm.validateRequest,
				},
				{
					Name: stepCheckActivateWallet,
					Func: fsm.checkActivateWallet,
				},
				{
					Name: stepBeforeActivationCheckTransferKind,
					Func: fsm.beforeActivationCheckTransferKind,
				},
				{
					Name:     stepActiveWalletResources,
					Func:     fsm.activateWalletResources,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepWaitingResourcesActivateWalletConfirmations,
					Func: fsm.waitingResourcesActivateWalletConfirmations,
				},
				{
					Name: stepWaitingExternalActivateWalletConfirmations,
					Func: fsm.waitingExternalActivateWalletConfirmations,
				},
				{
					Name:     stepActivateWalletBurnTRX,
					Func:     fsm.activateWalletBurnTRX,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepWaitingActivateWalletConfirmations,
					Func: fsm.waitingActivateWalletConfirmations,
				},
				{
					Name: stepBeforeSendingCheckTransferKind,
					Func: fsm.beforeSendingCheckTransferKind,
				},
				{
					Name: stepSendTRXForBurn,
					Func: fsm.sendTRXForBurn,
				},
				{
					Name: stepWaitingSendTRXForBurnConfirmations,
					Func: fsm.waitingSendBurnTrxConfirmations,
				},
				{
					Name:     stepDelegateResources,
					Func:     fsm.delegateResources,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepWaitingDelegateConfirmations,
					Func: fsm.waitingDelegateConfirmations,
				},
				{
					Name: stepWaitingExternalDelegateConfirmations,
					Func: fsm.waitingExternalDelegateResources,
				},
			},
		},
		{
			Name: stageSending,
			Steps: []*workflow.Step{
				{
					Name: stepSending,
					Func: fsm.sending,
				},
			},
		},
		{
			Name: stageAfterSending,
			Steps: []*workflow.Step{
				{
					Name:     stepWaitingForTheFirstConfirmation,
					Func:     fsm.waitingForTheFirstConfirmation,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name:     stepWaitingConfirmations,
					Func:     fsm.waitingConfirmations,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepAfterSendingCheckTransferKind,
					Func: fsm.afterSendingCheckTransferKind,
				},
				{
					Name: stepReclaimResources,
					Func: fsm.reclaimResources,
				},
				{
					Name:     stepWaitingReclaimRresourcesConfirmations,
					Func:     fsm.waitingReclaimResourcesConfirmations,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepSendSuccessEvent,
					Func: fsm.sendSuccessEvent,
				},
			},
		},
		// Compensation stage (will never be reached in success scenario)
		{
			Name: stageCompensateOnFail,
			Steps: []*workflow.Step{
				{
					Name: stepDetermineCompensationFlow,
					Func: fsm.determineCompensationFlow,
				},
				{
					Name: stepReclaimOnError,
					Func: fsm.reclaimOnError,
				},
				{
					Name: stepWaitingReclaimOnErrorConfirmation,
					Func: fsm.waitingReclaimOnErrorConfirmations,
				},
				{
					Name: stepSendFailureEvent,
					Func: fsm.sendFailureEvent,
				},
			},
		},
	})

	// set snapshot for the workflow
	if err := fsm.wf.SetSnapshot(transfer.WorkflowSnapshot); err != nil {
		return nil, fmt.Errorf("set workflow snapshot for transfer %s: %w", transfer.ID.String(), err)
	}

	return fsm, nil
}

// Run executes the workflow.
func (s *FSM) Run(ctx context.Context) error {
	return s.wf.Run(
		constants.WithClientContext(ctx, s.transfer.ClientID),
	)
}
