package system

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/dv-net/dv-processing/internal/constants"

	"github.com/dv-net/dv-processing/sql"
	"github.com/jackc/pgx/v5"
)

// CheckDBVersion checks if the database version is equal to the last migration version.
func (s service) CheckDBVersion(ctx context.Context) error {
	files, err := sql.MigrationsFS.ReadDir(sql.PostgresMigrationParams().Path)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	migrationVersions := make([]uint64, 0, len(files))

	for _, file := range files {
		parts := strings.SplitN(file.Name(), "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid migration file name: %s", file.Name())
		}

		version, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return err
		}

		migrationVersions = append(migrationVersions, version)
	}

	slices.Sort(migrationVersions)

	dbver, err := s.store.System().GetMigrationVersion(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("no migration in database")
		}
		return err
	}

	if dbver != migrationVersions[len(migrationVersions)-1] {
		return fmt.Errorf(
			"db version %d is not equal to last migration version %d",
			dbver, migrationVersions[len(migrationVersions)-1],
		)
	}

	return nil
}

// SystemVersion get system version
func (s service) SystemVersion(_ context.Context) string {
	return s.version
}

// SystemCommit get system commit
func (s service) SystemCommit(_ context.Context) string {
	return s.commit
}

// ProcessingID get processing identity
func (s service) ProcessingID(ctx context.Context) (string, error) {
	name := constants.ProcessingIDParamName
	setting, ok := s.store.Cache().GlobalSettings().Load(name.String())
	if ok {
		return setting.Value, nil
	}

	var err error
	setting, err = s.store.Settings().GetGlobalByName(ctx, name.String())
	if err != nil {
		return "", fmt.Errorf("fetch processing id: %w", err)
	}

	s.store.Cache().GlobalSettings().Store(setting.Name, setting)

	return setting.Value, nil
}

// SetDvSecretKey set dv admin secret key
func (s service) SetDvSecretKey(ctx context.Context, secret string) error {
	set, err := s.store.Settings().SetGlobal(ctx, constants.DVAdminSecretKeyName, secret)
	if err != nil {
		return fmt.Errorf("set processing id: %w", err)
	}

	s.store.Cache().GlobalSettings().Store(constants.DVAdminSecretKeyName, set)

	return nil
}
