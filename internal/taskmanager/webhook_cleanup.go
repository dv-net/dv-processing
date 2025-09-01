package taskmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"

	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/robfig/cron/v3"
)

const (
	WebhookCleanupPeriodicJob = "webhook_cleanup"
)

func getWebhookCleanupJob(cronRule string) (*river.PeriodicJob, error) {
	s, err := cron.ParseStandard(cronRule)
	if err != nil {
		return nil, err
	}

	return river.NewPeriodicJob(s, func() (river.JobArgs, *river.InsertOpts) {
		return WebhookCleanupJobArgs{}, nil
	}, &river.PeriodicJobOpts{
		RunOnStart: true, // Set once
	}), nil
}

type WebhookCleanupJobArgs struct{}

func (WebhookCleanupJobArgs) Kind() string { return WebhookCleanupPeriodicJob }

type WebhookCleanupWorker struct {
	river.WorkerDefaults[WebhookCleanupJobArgs]

	logger logger.Logger
	config *config.Config

	store store.IStore
	bs    baseservices.IBaseServices
}

func (s *WebhookCleanupWorker) Timeout(*river.Job[WebhookCleanupJobArgs]) time.Duration {
	return -1
}

func (s *WebhookCleanupWorker) Work(ctx context.Context, _ *river.Job[WebhookCleanupJobArgs]) error {
	affectedRows, err := s.store.Webhooks().Cleanup(ctx, pgtype.Timestamptz{
		Time:  time.Now().Add(-s.config.Webhooks.Cleanup.MaxAge),
		Valid: true,
	})
	if err != nil {
		return fmt.Errorf("failed to delete old webhooks: %w", err)
	}

	if affectedRows > 0 {
		s.logger.Infof("deleted %d old compledted webhooks", affectedRows)
	}

	return nil
}
