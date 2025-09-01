package taskmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/fsm/fsmbch"
	"github.com/dv-net/dv-processing/internal/fsm/fsmbtc"
	"github.com/dv-net/dv-processing/internal/fsm/fsmdoge"
	"github.com/dv-net/dv-processing/internal/fsm/fsmevm"
	"github.com/dv-net/dv-processing/internal/fsm/fsmltc"
	"github.com/dv-net/dv-processing/internal/fsm/fsmtron"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/logger"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const JobKindTransferWorkflow = "transfer_workflow"

type TransferWorkflowArgs struct {
	TransferID uuid.UUID `json:"transfer_id"`
}

// Kind
func (TransferWorkflowArgs) Kind() string { return JobKindTransferWorkflow }

// func (args TransferWorkflowArgs) InsertOpts() river.InsertOpts {
// 	return river.InsertOpts{MaxAttempts: 1}
// }

type TransferWorkflowWorker struct {
	river.WorkerDefaults[TransferWorkflowArgs]

	logger logger.Logger
	config *config.Config

	store store.IStore
	bs    baseservices.IBaseServices
}

func (s *TransferWorkflowWorker) Timeout(*river.Job[TransferWorkflowArgs]) time.Duration { return -1 }

func (s *TransferWorkflowWorker) Work(ctx context.Context, job *river.Job[TransferWorkflowArgs]) error {
	transfer, err := s.bs.Transfers().GetByID(ctx, job.Args.TransferID)
	if err != nil {
		return err
	}

	handlers := map[wconstants.BlockchainType]func(ctx context.Context, transfer *models.Transfer) error{
		wconstants.BlockchainTypeTron: s.handleTronFSM,

		wconstants.BlockchainTypeEthereum:          s.handleEvmFSM,
		wconstants.BlockchainTypeBinanceSmartChain: s.handleEvmFSM,
		wconstants.BlockchainTypePolygon:           s.handleEvmFSM,
		wconstants.BlockchainTypeArbitrum:          s.handleEvmFSM,
		wconstants.BlockchainTypeOptimism:          s.handleEvmFSM,
		wconstants.BlockchainTypeLinea:             s.handleEvmFSM,

		wconstants.BlockchainTypeBitcoin:     s.handleBtcFSM,
		wconstants.BlockchainTypeLitecoin:    s.handleLtcFSM,
		wconstants.BlockchainTypeBitcoinCash: s.handleBchFSM,
		wconstants.BlockchainTypeDogecoin:    s.handleDogeFSM,
	}

	handler, ok := handlers[transfer.Blockchain]
	if !ok {
		return fmt.Errorf("unsupported blockchain type: %s", transfer.Blockchain)
	}

	return handler(ctx, transfer)
}

func (s *TransferWorkflowWorker) handleBtcFSM(ctx context.Context, transfer *models.Transfer) error {
	if !s.config.Blockchain.Bitcoin.Enabled {
		return fmt.Errorf("bitcoin blockchain is disabled in config: %w", river.JobSnooze(time.Minute))
	}

	fsm, err := fsmbtc.NewFSM(s.logger, s.config, s.store, s.bs, transfer)
	if err != nil {
		return fmt.Errorf("create btc fsm: %w", err)
	}

	return fsm.Run(ctx)
}

func (s *TransferWorkflowWorker) handleLtcFSM(ctx context.Context, transfer *models.Transfer) error {
	if !s.config.Blockchain.Bitcoin.Enabled {
		return fmt.Errorf("litecoin is disabled in config: %w", river.JobSnooze(time.Minute))
	}

	fsm, err := fsmltc.NewFSM(s.logger, s.config, s.store, s.bs, transfer)
	if err != nil {
		return fmt.Errorf("create ltc fsm: %w", err)
	}

	return fsm.Run(ctx)
}

func (s *TransferWorkflowWorker) handleBchFSM(ctx context.Context, transfer *models.Transfer) error {
	if !s.config.Blockchain.BitcoinCash.Enabled {
		return fmt.Errorf("bitcoin cash is disabled in config: %w", river.JobSnooze(time.Minute))
	}

	fsm, err := fsmbch.NewFSM(s.logger, s.config, s.store, s.bs, transfer)
	if err != nil {
		return fmt.Errorf("create bch fsm: %w", err)
	}

	return fsm.Run(ctx)
}

func (s *TransferWorkflowWorker) handleDogeFSM(ctx context.Context, transfer *models.Transfer) error {
	if !s.config.Blockchain.Dogecoin.Enabled {
		return fmt.Errorf("dogecoin is disabled in config: %w", river.JobSnooze(time.Minute))
	}

	fsm, err := fsmdoge.NewFSM(s.logger, s.config, s.store, s.bs, transfer)
	if err != nil {
		return fmt.Errorf("create doge fsm: %w", err)
	}

	return fsm.Run(ctx)
}

func (s *TransferWorkflowWorker) handleTronFSM(ctx context.Context, transfer *models.Transfer) error {
	if !s.config.Blockchain.Tron.Enabled {
		return fmt.Errorf("tron blockchain is disabled in config: %w", river.JobSnooze(time.Minute))
	}

	if s.bs.RManager() == nil && transfer.Kind.String == string(constants.TronTransferKindCloudDelegate) {
		return fmt.Errorf("resource manager is not initialized: %w", river.JobSnooze(time.Minute))
	}

	if s.bs.Tron() == nil {
		return fmt.Errorf("tron service is not initialized: %w", river.JobSnooze(time.Minute))
	}

	fsm, err := fsmtron.NewFSM(s.logger, s.config, s.store, s.bs, transfer)
	if err != nil {
		return fmt.Errorf("create tron fsm: %w", err)
	}

	return fsm.Run(ctx)
}

func (s *TransferWorkflowWorker) handleEvmFSM(ctx context.Context, transfer *models.Transfer) error {
	evmInstance, err := s.bs.Blockchains().GetEVMByBlockchain(transfer.Blockchain)
	if err != nil {
		return fmt.Errorf("get %s instance: %w", transfer.Blockchain.String(), err)
	}

	evmConfig, err := s.config.Blockchain.GetEVMByBlockchainType(transfer.Blockchain)
	if err != nil {
		return fmt.Errorf("get %s config: %w", transfer.Blockchain.String(), err)
	}

	if !evmConfig.IsEnabled() {
		return fmt.Errorf("%s blockchain is disabled in config: %w", transfer.Blockchain.String(), river.JobSnooze(time.Minute))
	}

	fsm, err := fsmevm.NewFSM(s.logger, evmConfig, s.config.IsEnabledSeedEncryption(), s.store, s.bs, evmInstance, transfer)
	if err != nil {
		return fmt.Errorf("create %s fsm: %w", transfer.Blockchain.String(), err)
	}

	return fsm.Run(ctx)
}
