package repo_system

import (
	"context"
)

func (s *CustomQuerier) GetMigrationVersion(ctx context.Context) (uint64, error) {
	row := s.db.QueryRow(ctx, `select version from schema_migrations order by version desc limit 1;`)
	var res uint64
	err := row.Scan(&res)
	return res, err
}
