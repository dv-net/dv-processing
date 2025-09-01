package webhooks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_webhooks"
	"github.com/jackc/pgx/v5/pgconn"
)

type BatchCreateParams = repo_webhooks.CreateParams

// BatchCreate creates multiple webhooks in batch.
func (s *Service) BatchCreate(ctx context.Context, params []BatchCreateParams, opts ...repos.Option) error {
	for _, p := range params {
		if p.Kind == "" {
			return fmt.Errorf("webhook kind is required")
		}

		if !p.Status.Valid() {
			return fmt.Errorf("status %s is invalid", p.Status)
		}
	}

	result := s.store.Webhooks(opts...).Create(ctx, params)

	wg := new(sync.WaitGroup)
	wg.Add(len(params))
	errCh := make(chan error, len(params))

	result.Exec(func(_ int, err error) {
		errCh <- err
		wg.Done()
	})

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return fmt.Errorf("batch create webhooks error: %w / %s", err, pgErr.Detail)
			}

			return fmt.Errorf("batch create webhooks error: %w", err)
		}
	}

	return nil
}
