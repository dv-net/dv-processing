package store

import (
	"github.com/dv-net/dv-processing/internal/store/cache"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IStore interface {
	repos.IRepos
	PSQLConn() *pgxpool.Pool
	Cache() cache.ICache
}

type store struct {
	repos.IRepos
	psql  *pgxpool.Pool
	cache cache.ICache
}

func New(
	psql *postgres.Postgres,
) IStore {
	return &store{
		IRepos: repos.New(psql),
		psql:   psql.DB,
		cache:  cache.New(),
	}
}

func (s store) PSQLConn() *pgxpool.Pool { return s.psql }
func (s store) Cache() cache.ICache     { return s.cache }
