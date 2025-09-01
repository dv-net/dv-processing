package postgres

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

type logger interface {
	Infof(string, ...any)
}

type Postgres struct {
	DB *pgxpool.Pool
}

func New(ctx context.Context, cfg Config, l logger) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(
		postgresDSN(cfg.Addr, cfg.User, url.QueryEscape(cfg.Password), cfg.DBName),
	)
	if err != nil {
		return nil, err
	}

	if cfg.MinConns > 0 && cfg.MinConns < cfg.MaxConns {
		config.MinConns = cfg.MinConns
		config.MaxConns = cfg.MaxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	l.Infof("successfully connected to postgres")

	return &Postgres{pool}, nil
}

func postgresDSN(addr, user, password, dbname string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		user, password, addr, dbname,
	)
}
