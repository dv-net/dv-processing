package sql

import (
	"embed"
)

type MigrationParameters struct {
	EmbedFs embed.FS
	Path    string
}

//go:embed postgres/migrations/*.sql
var MigrationsFS embed.FS

func PostgresMigrationParams() MigrationParameters {
	return MigrationParameters{
		Path:    "postgres/migrations",
		EmbedFs: MigrationsFS,
	}
}
