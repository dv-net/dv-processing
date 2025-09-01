package transfers

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfers"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/internal/workflow"
	"github.com/dv-net/dv-processing/pkg/dbutils/pgtypeutils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetByID returns the transfer by the ID.
func (s *Service) GetByID(ctx context.Context, transferID uuid.UUID) (*models.Transfer, error) {
	if transferID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	data, err := s.store.Transfers().GetByID(ctx, transferID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

func (s *Service) GetByRequestID(ctx context.Context, requestID string) (*models.Transfer, error) {
	if requestID == "" {
		return nil, storecmn.ErrEmptyID
	}

	data, err := s.store.Transfers().GetByRequestID(ctx, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

func (s *Service) GetSystemTransactionsByTransfer(ctx context.Context, transferID uuid.UUID) ([]*models.TransferTransaction, error) {
	return s.store.TransferTransactions().GetByTransfer(ctx, transferID)
}

// GetByTxHashAndOwnerID returns the transfer by the txHash and ownerID.
func (s *Service) GetByTxHashAndOwnerID(ctx context.Context, txHash string, ownerID uuid.UUID) (*models.Transfer, error) {
	if txHash == "" {
		return nil, storecmn.ErrEmptyHash
	}

	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	data, err := s.store.Transfers().GetByTxHashAndOwnerID(ctx, pgtypeutils.EncodeText(&txHash), ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// ExistsByTxHashAndOwnerID checks if the transfer with the specified txHash and ownerID exists.
func (s *Service) ExistsByTxHashAndOwnerID(ctx context.Context, txHash string, ownerID uuid.UUID) (bool, error) {
	if txHash == "" {
		return false, storecmn.ErrEmptyHash
	}

	if ownerID == uuid.Nil {
		return false, storecmn.ErrEmptyID
	}

	return s.store.Transfers().ExistsByTxHashAndOwnerID(ctx, pgtypeutils.EncodeText(&txHash), ownerID)
}

// FindAllNewTransfers returns all new transfers.
func (s *Service) FindAllNewTransfers(ctx context.Context) ([]*models.Transfer, error) {
	return s.store.Transfers().FindAllNewTransfers(ctx)
}

// SetTxHash sets the txHash for the transfer.
func (s *Service) SetTxHash(ctx context.Context, transferID uuid.UUID, txHash string, opts ...repos.Option) (*models.Transfer, error) {
	if transferID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	if txHash == "" {
		return nil, storecmn.ErrEmptyHash
	}

	return s.store.Transfers(opts...).SetTxHash(ctx, transferID, pgtypeutils.EncodeText(&txHash))
}

// SetStatus sets the status for the transfer.
func (s *Service) SetStatus(ctx context.Context, transferID uuid.UUID, status constants.TransferStatus, opts ...repos.Option) error {
	if transferID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if !status.Valid() {
		return fmt.Errorf("status %s is invalid", status)
	}

	return s.store.Transfers(opts...).SetStatus(ctx, transferID, status)
}

// SetWorkflowSnapshot sets the workflow snapshot for the transfer.
func (s *Service) SetWorkflowSnapshot(ctx context.Context, transferID uuid.UUID, snapshot workflow.Snapshot, opts ...repos.Option) error {
	if transferID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	return s.store.Transfers(opts...).SetWorkflowSnapshot(ctx, transferID, snapshot)
}

// GetWorkflowSnapshot returns the workflow snapshot for the transfer.
func (s *Service) GetWorkflowSnapshot(ctx context.Context, transferID uuid.UUID) (*workflow.Snapshot, error) {
	if transferID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	snapshot, err := s.store.Transfers().GetWorkflowSnapshot(ctx, transferID)
	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// SetStateData sets the state data for the transfer.
func (s *Service) SetStateData(ctx context.Context, transferID uuid.UUID, stateData map[string]any, opts ...repos.Option) error {
	if transferID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	if stateData == nil {
		return fmt.Errorf("state data is required")
	}

	// get current state data
	currentStateData, err := s.GetStateData(ctx, transferID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	// merge state data
	for k, v := range stateData {
		currentStateData[k] = v
	}

	return s.store.Transfers(opts...).SetStateData(ctx, transferID, currentStateData)
}

// GetStateData returns the state data for the transfer.
func (s *Service) GetStateData(ctx context.Context, transferID uuid.UUID) (map[string]any, error) {
	if transferID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	data, err := s.store.Transfers().GetStateData(ctx, transferID)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type FindParams = repo_transfers.FindParams

// Find
func (s *Service) Find(ctx context.Context, params FindParams) ([]*models.Transfer, error) {
	return s.store.Transfers().Find(ctx, params)
}

// GetActiveTronTransfersResources
func (s *Service) GetActiveTronTransfersResources(ctx context.Context) (*repo_transfers.GetActiveTronTransfersResourcesRow, error) {
	return s.store.Transfers().GetActiveTronTransfersResources(ctx)
}

// GetActiveTronTransfersBurn
func (s *Service) GetActiveTronTransfersBurn(ctx context.Context) (*repo_transfers.GetActiveTronTransfersBurnRow, error) {
	return s.store.Transfers().GetActiveTronTransfersBurn(ctx)
}
