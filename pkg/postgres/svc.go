package postgres

import "context"

func (s *Postgres) Name() string { return "postgres" }

func (s *Postgres) Start(_ context.Context) error { return nil }

func (s *Postgres) Stop(_ context.Context) error {
	s.DB.Close()
	return nil
}

func (s *Postgres) Ping(ctx context.Context) error { return s.DB.Ping(ctx) }
