package taskmanager

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type TaskManager struct {
	riverClient *river.Client[pgx.Tx]
}

// New
func New(
	l logger.ExtendedLogger,
	conf *config.Config,
	st store.IStore,
	bs baseservices.IBaseServices,
) (*TaskManager, error) {
	var periodicJobs []*river.PeriodicJob

	workers := river.NewWorkers()

	river.AddWorker(workers, &WebhookWaitingConfirmationsWorker{
		logger: l,
		bs:     bs,
	})

	river.AddWorker(workers, &TransferWorkflowWorker{
		logger: l,
		config: conf,
		store:  st,
		bs:     bs,
	})

	if conf.Webhooks.Cleanup.Enabled {
		river.AddWorker(workers, &WebhookCleanupWorker{
			logger: l,
			config: conf,
			store:  st,
			bs:     bs,
		})

		cr, err := getWebhookCleanupJob(conf.Webhooks.Cleanup.Cron)
		if err != nil {
			return nil, fmt.Errorf("webhook cleanup job: %w", err)
		}

		periodicJobs = append(periodicJobs, cr)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(st.PSQLConn()), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
		},
		Workers:      workers,
		Logger:       newLogger(l),
		PeriodicJobs: periodicJobs,
	})
	if err != nil {
		return nil, fmt.Errorf("init river client: %w", err)
	}

	return &TaskManager{
		riverClient: riverClient,
	}, nil
}

// Name
func (s *TaskManager) Name() string { return "task-manager" }

// Start
func (s *TaskManager) Start(ctx context.Context) error {
	if s.riverClient == nil {
		return nil
	}
	return s.riverClient.Start(ctx)
}

// Stop
func (s *TaskManager) Stop(ctx context.Context) error {
	if s.riverClient == nil {
		return nil
	}
	return s.riverClient.Stop(ctx)
}

// Client
func (s *TaskManager) Client() *river.Client[pgx.Tx] { return s.riverClient }
