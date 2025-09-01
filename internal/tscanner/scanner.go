package tscanner

import (
	"context"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

var scanInterval = 1 * time.Second

// processAllNewTransfers processes all new transfers.
func (s *TScanner) processAllNewTransfers(ctx context.Context) error {
	if !s.inUse.CompareAndSwap(false, true) {
		return nil
	}
	defer s.inUse.Store(false)

	newTransfers, err := s.bs.Transfers().FindAllNewTransfers(ctx)
	if err != nil {
		return err
	}

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, transfer := range newTransfers {
			if err := s.handleTransfer(ctx, tx, transfer); err != nil {
				return fmt.Errorf("handle transfer: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	return nil
}

// handleTransfer adds a transfer to the task manager and sets the transfer status to processing.
func (s *TScanner) handleTransfer(ctx context.Context, dbTx pgx.Tx, transfer *models.Transfer) error {
	_, err := s.tm.Client().InsertTx(ctx, dbTx,
		taskmanager.TransferWorkflowArgs{
			TransferID: transfer.ID,
		},
		&river.InsertOpts{
			// UniqueOpts: river.UniqueOpts{
			// 	ByArgs: true,
			// },
		},
	)
	if err != nil {
		return fmt.Errorf("insert transfer workflow job: %w", err)
	}

	if err := s.bs.Transfers().SetStatus(ctx, transfer.ID, constants.TransferStatusProcessing, repos.WithTx(dbTx)); err != nil {
		return fmt.Errorf("set transfer status: %w", err)
	}

	return nil
}
