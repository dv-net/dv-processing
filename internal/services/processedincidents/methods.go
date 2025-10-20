package processedincidents

import (
	"context"
	"errors"
	"fmt"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_processed_incidents"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// IsProcessed checks if an incident has been processed
func (s *Service) IsProcessed(ctx context.Context, blockchain wconstants.BlockchainType, incidentID string, opts ...repos.Option) (bool, error) {
	if blockchain == "" {
		return false, errBlockchainEmpty
	}

	if incidentID == "" {
		return false, errIncidentIDEmpty
	}

	exists, err := s.store.ProcessedIncidents(opts...).IsIncidentProcessed(ctx, blockchain, incidentID)
	if err != nil {
		return false, fmt.Errorf("check if incident processed: %w", err)
	}

	return exists, nil
}

// MarkAsProcessing marks an incident as being processed
func (s *Service) MarkAsProcessing(ctx context.Context, blockchain wconstants.BlockchainType, incidentID string, incidentType string, rollbackFromBlock, rollbackToBlock int64, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if incidentID == "" {
		return errIncidentIDEmpty
	}

	return s.store.ProcessedIncidents(opts...).MarkIncidentAsProcessing(ctx, repo_processed_incidents.MarkIncidentAsProcessingParams{
		ID:                incidentID,
		Blockchain:        blockchain,
		IncidentType:      incidentType,
		RollbackFromBlock: pgtype.Int8{Int64: rollbackFromBlock, Valid: true},
		RollbackToBlock:   pgtype.Int8{Int64: rollbackToBlock, Valid: true},
	})
}

// MarkAsCompleted marks an incident as successfully completed
func (s *Service) MarkAsCompleted(ctx context.Context, blockchain wconstants.BlockchainType, incidentID string, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if incidentID == "" {
		return errIncidentIDEmpty
	}

	return s.store.ProcessedIncidents(opts...).MarkIncidentAsCompleted(ctx, blockchain, incidentID)
}

// MarkAsFailed marks an incident as failed with an error message
func (s *Service) MarkAsFailed(ctx context.Context, blockchain wconstants.BlockchainType, incidentID string, errorMessage string, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if incidentID == "" {
		return errIncidentIDEmpty
	}

	return s.store.ProcessedIncidents(opts...).MarkIncidentAsFailed(ctx, repo_processed_incidents.MarkIncidentAsFailedParams{
		Blockchain:   blockchain,
		ID:           incidentID,
		ErrorMessage: pgtype.Text{String: errorMessage, Valid: true},
	})
}

// GetIncompleteIncidents returns all incidents that are still being processed
func (s *Service) GetIncompleteIncidents(ctx context.Context, blockchain wconstants.BlockchainType, opts ...repos.Option) ([]*models.ProcessedIncident, error) {
	if blockchain == "" {
		return nil, errBlockchainEmpty
	}

	incidents, err := s.store.ProcessedIncidents(opts...).GetIncompleteIncidents(ctx, blockchain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*models.ProcessedIncident{}, nil
		}
		return nil, fmt.Errorf("get incomplete incidents: %w", err)
	}

	return incidents, nil
}

// CleanupOldIncidents removes old completed incidents (older than 30 days)
func (s *Service) CleanupOldIncidents(ctx context.Context, opts ...repos.Option) error {
	return s.store.ProcessedIncidents(opts...).CleanupOldIncidents(ctx)
}
