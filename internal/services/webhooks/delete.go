package webhooks

import (
	"context"

	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/google/uuid"
)

// Delete - deletes a webhook by its ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	return s.store.Webhooks().DeleteByID(ctx, id)
}
