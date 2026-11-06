package escanner

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// incidentJobStatesQuery counts the backfill jobs of an incident grouped by
// river state. excludeJobID excludes a job from the count (the worker's own
// job is still running while it performs the completion check).
const incidentJobStatesQuery = `
SELECT state::text, count(*)
FROM river_job
WHERE kind = $1 AND args->>'incident_id' = $2 AND id != $3
GROUP BY state
`

// activeIncidentJobStates are river states meaning the backfill is still in flight.
var activeIncidentJobStates = []string{"available", "pending", "running", "retryable", "scheduled"}

type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type incidentJobCounts map[string]int64

func (c incidentJobCounts) active() int64 {
	var n int64
	for _, state := range activeIncidentJobStates {
		n += c[state]
	}
	return n
}

func (c incidentJobCounts) total() int64 {
	var n int64
	for _, count := range c {
		n += count
	}
	return n
}

func loadIncidentJobCounts(ctx context.Context, db dbQuerier, incidentID string, excludeJobID int64) (incidentJobCounts, error) {
	rows, err := db.Query(ctx, incidentJobStatesQuery, JobKindAddressBackfill, incidentID, excludeJobID)
	if err != nil {
		return nil, fmt.Errorf("query incident jobs: %w", err)
	}
	defer rows.Close()

	counts := make(incidentJobCounts)
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return nil, fmt.Errorf("scan incident job count: %w", err)
		}
		counts[state] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incident job counts: %w", err)
	}

	return counts, nil
}

// reconcileIncidents closes out address_backfill incidents left over from
// previous runs. River resumes pending jobs on its own, so this only marks
// finished incidents as completed and surfaces incidents with discarded jobs.
func (s *scanner) reconcileIncidents(ctx context.Context) error {
	incidents, err := s.bs.ProcessedIncidents().GetIncompleteIncidents(ctx, s.blockchain)
	if err != nil {
		return fmt.Errorf("get incomplete incidents: %w", err)
	}

	for _, incident := range incidents {
		if incident.IncidentType != IncidentTypeAddressBackfill {
			continue
		}

		counts, err := loadIncidentJobCounts(ctx, s.store.PSQLConn(), incident.ID, 0)
		if err != nil {
			return fmt.Errorf("count jobs for incident %s: %w", incident.ID, err)
		}

		switch {
		case counts.active() > 0:
			s.logger.Infof("incident %s: backfill in progress (%d active jobs)", incident.ID, counts.active())
		case counts["discarded"] > 0:
			s.logger.Errorf("incident %s: %d backfill jobs were discarded, manual requeue required", incident.ID, counts["discarded"])
		default:
			// everything is completed or cancelled — or nothing was enqueued at all
			if err := s.bs.ProcessedIncidents().MarkAsCompleted(ctx, s.blockchain, incident.ID); err != nil {
				return fmt.Errorf("complete incident %s: %w", incident.ID, err)
			}
			s.logger.Infof("incident %s completed: no active backfill jobs left", incident.ID)
		}
	}

	return nil
}
