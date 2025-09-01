package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4/database"
	postgresdb "github.com/golang-migrate/migrate/v4/database/postgres"
)

func (s *Migration) postgresDriver(ctx context.Context) (database.Driver, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		s.config.User,
		url.QueryEscape(s.config.Password),
		s.config.Addr,
		s.config.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open databse: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping databse: %w", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("databse connection: %w", err)
	}

	instance, err := postgresdb.WithConnection(
		ctx, conn,
		&postgresdb.Config{
			MigrationsTable: "schema_migrations",
			DatabaseName:    s.config.DBName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("init postgres instance error: %w", err)
	}

	return instance, nil
}
