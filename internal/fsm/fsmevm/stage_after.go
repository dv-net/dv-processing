package fsmevm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/ethereum/go-ethereum/common"
	"github.com/riverqueue/river"
)

// waitingForTheFirstConfirmation
// TODO: in current logic transfer will be failed when got stuck in mempool as long as system tx are pended
// TODO: FIX IN: DV-2362
func (s *FSM) waitingForTheFirstConfirmation(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	_, isPending, err := s.evm.Node().TransactionByHash(ctx, common.HexToHash(s.transfer.TxHash.String))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") &&
			s.transfer.CreatedAt.Valid &&
			time.Since(s.transfer.CreatedAt.Time) > time.Minute*5 {
			// Transaction has been removed from blockchain
			return newErrFailedTransfer(fmt.Errorf("transaction %s not found in the blockchain: %w", s.transfer.TxHash.String, err))
		}
		return err
	}

	// if transaction is pending, snooze for 1 second
	if isPending {
		duration := time.Second
		if s.transfer.CreatedAt.Valid && time.Since(s.transfer.CreatedAt.Time) > 1*time.Hour {
			duration = 30 * time.Second
		}
		return workflow.NoConsoleError(river.JobSnooze(duration))
	}

	// check confirmations
	if err = s.ensureTxInBlockchain(ctx, s.transfer.TxHash.String); err != nil {
		return err
	}

	return s.setTransferStatus(ctx, constants.TransferStatusUnconfirmed)
}

// waitingConfirmations
func (s *FSM) waitingConfirmations(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	// check confirmations
	if err := s.checkTransactionConfirmations(ctx, s.transfer.TxHash.String, constants.GetMinConfirmations(s.evm.Blockchain())); err != nil {
		return err
	}

	return nil
}

// sendSuccessEvent
func (s *FSM) sendSuccessEvent(ctx context.Context, _ *workflow.Workflow, _ *workflow.Stage, _ *workflow.Step) error {
	if err := s.setTransferStatus(ctx, constants.TransferStatusCompleted); err != nil {
		return err
	}
	return nil
}
