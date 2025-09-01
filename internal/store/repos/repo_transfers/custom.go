package repo_transfers

import (
	"context"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/jackc/pgx/v5"
)

type ICustomQuerier interface {
	Querier
	Find(ctx context.Context, params FindParams) ([]*models.Transfer, error)
}

var _ ICustomQuerier = (*CustomQuerier)(nil)

type CustomQuerier struct {
	*Queries
	psql DBTX
}

func NewCustom(psql DBTX) *CustomQuerier {
	return &CustomQuerier{
		Queries: New(psql),
		psql:    psql,
	}
}

func (s *CustomQuerier) WithTx(tx pgx.Tx) *CustomQuerier {
	return &CustomQuerier{
		Queries: New(tx),
		psql:    tx,
	}
}
