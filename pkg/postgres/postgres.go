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
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	// cfg values may be unset when the config loader did not apply defaults;
	// fall back to the documented defaults instead of the pgxpool defaults
	// (max(4, NumCPU)), which depend on the machine and are too small for the
	// river workers (up to 50 concurrent jobs).
	minConns, maxConns := cfg.MinConns, cfg.MaxConns
	if maxConns <= 0 {
		maxConns = 30
	}
	if minConns <= 0 || minConns >= maxConns {
		minConns = 5
	}
	config.MinConns = minConns
	config.MaxConns = maxConns

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
		"postgres://%s:%s@%s/%s?sslmode=disable&application_name=dv-processing",
		user, password, addr, dbname,
	)
}
