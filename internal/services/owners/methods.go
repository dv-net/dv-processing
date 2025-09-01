package owners

import (
	"context"
	"errors"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetAll returns all owners.
func (s *Service) GetAll(ctx context.Context) ([]*models.Owner, error) {
	return s.store.Owners().GetAll(ctx)
}

// GetByID returns the owner by the ID.
func (s *Service) GetByID(ctx context.Context, ownerID uuid.UUID) (*models.Owner, error) {
	if ownerID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	item := s.store.Cache().Owners().Get(ownerID.String())
	if item != nil {
		return item.Value(), nil
	}

	data, err := s.store.Owners().GetByID(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	s.store.Cache().Owners().Set(ownerID.String(), data, 0)

	return data, nil
}
