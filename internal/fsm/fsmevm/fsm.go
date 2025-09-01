package fsmevm

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/store/repos"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
)

type FSM struct {
	logger                logger.Logger
	config                config.IEVMConfig
	enabledSeedEncryption bool
	wf                    *workflow.Workflow
	transfer              *models.Transfer
	st                    store.IStore

	evm *evm.EVM

	// services
	bs baseservices.IBaseServices
}

func NewFSM(
	l logger.Logger,
	conf config.IEVMConfig,
	enabledSeedCompression bool,
	st store.IStore,
	bs baseservices.IBaseServices,
	evmInstance *evm.EVM,
	transfer *models.Transfer,
) (*FSM, error) {
	fsm := &FSM{
		logger:                l,
		config:                conf,
		enabledSeedEncryption: enabledSeedCompression,
		st:                    st,
		bs:                    bs,
		evm:                   evmInstance,
		transfer:              transfer,
	}

	// create a workflow
	fsm.wf = workflow.New(
		workflow.WithName(evmInstance.Blockchain().String()+" fsm"),
		workflow.WithLogger(l),
		workflow.WithDebug(true),
		workflow.WithBeforeAllStepsFn(func(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
			if err := fsm.bs.Transfers().SetWorkflowSnapshot(ctx, fsm.transfer.ID, fsm.wf.GetSnapshot()); err != nil {
				return fmt.Errorf("set workflow snapshot: %w", err)
			}

			return nil
		}),
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

		if errors.Is(err, errFailedTransfer) {
			return pgx.BeginTxFunc(ctx, st.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
				return fsm.sendFailureEvent(ctx, w, err, repos.WithTx(tx))
			})
		}

		if w.CurrentStage() != nil && w.CurrentStage().Name != "" &&
			w.CurrentStep() != nil && w.CurrentStep().Name != "" {
			currentStage, currentStep := w.CurrentStage().Name, w.CurrentStep().Name
			if steps, ok := exitOnFailedSteps[currentStage]; ok {
				for _, step := range steps {
					if step == currentStep {
						return pgx.BeginTxFunc(ctx, st.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
							return fsm.sendFailureEvent(ctx, w, err, repos.WithTx(tx))
						})
					}
				}
			}
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
					Name:     stepSendBaseAssetForBurn,
					Func:     fsm.sendBaseAssetForBurn,
					BeforeFn: fsm.createTransferStepWh,
				},
				{
					Name: stepWaitingSendBaseAssetForBurnConfirmations,
					Func: fsm.waitingSendBurnBaseAssetConfirmations,
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
					Name: stepSendSuccessEvent,
					Func: fsm.sendSuccessEvent,
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
