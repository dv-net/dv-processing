package main

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/config"

	"github.com/dv-net/dv-processing/pkg/migrations"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/mx/logger"
	"github.com/urfave/cli/v3"
)

type migrateConfig struct {
	Log      logger.Config   `yaml:"log"`
	Postgres postgres.Config `yaml:"postgres"`
}

func initMigrations(l logger.Logger, conf postgres.Config, disableConfiramtions bool) (*migrations.Migration, error) {
	return migrations.New(l, migrations.Config{
		DBDriver:            migrations.DBDriverPostgres,
		User:                conf.User,
		Password:            conf.Password,
		Addr:                conf.Addr,
		DBName:              conf.DBName,
		DisableConfirmation: disableConfiramtions,
	})
}

func migrateCMD() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "migration database schema",
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "up database schema",
				Flags: []cli.Flag{
					cfgPathsFlag(),
					&cli.IntFlag{
						Name:        "steps",
						Aliases:     []string{"s"},
						Usage:       "number of steps to migrate",
						DefaultText: "by default, all migrations are applied",
					},
					&cli.BoolFlag{
						Name:  "silent",
						Usage: "disable confirmation for migration",
					},
				},
				Action: func(ctx context.Context, cl *cli.Command) error {
					conf, err := config.Load[migrateConfig](cl.StringSlice("configs"), envPrefix)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					l := logger.NewExtended(append(defaultLoggerOpts(), logger.WithConfig(conf.Log))...)

					mg, err := initMigrations(l, conf.Postgres, cl.Bool("silent"))
					if err != nil {
						return fmt.Errorf("failed to init migrations: %w", err)
					}

					return mg.Up(ctx, cl.Int("steps"))
				},
			},
			{
				Name:  "down",
				Usage: "rollback database schema",
				Flags: []cli.Flag{
					cfgPathsFlag(),
					&cli.IntFlag{
						Name:        "steps",
						Aliases:     []string{"s"},
						Usage:       "number of steps to migrate",
						DefaultText: "by default, 1 migration is rolled back. use -1 to rollback all migrations",
						Value:       1,
					},
					&cli.BoolFlag{
						Name:  "silent",
						Usage: "disable confirmation for migration",
					},
				},
				Action: func(ctx context.Context, cl *cli.Command) error {
					conf, err := config.Load[migrateConfig](cl.StringSlice("configs"), envPrefix)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					l := logger.NewExtended(append(defaultLoggerOpts(), logger.WithConfig(conf.Log))...)

					mg, err := initMigrations(l, conf.Postgres, cl.Bool("silent"))
					if err != nil {
						return fmt.Errorf("failed to init migrations: %w", err)
					}

					return mg.Down(ctx, cl.Int("steps"))
				},
			},
			{
				Name:  "drop",
				Usage: "drop database schema",
				Flags: []cli.Flag{cfgPathsFlag()},
				Action: func(ctx context.Context, cl *cli.Command) error {
					conf, err := config.Load[migrateConfig](cl.StringSlice("configs"), envPrefix)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					l := logger.NewExtended(append(defaultLoggerOpts(), logger.WithConfig(conf.Log))...)

					mg, err := initMigrations(l, conf.Postgres, false)
					if err != nil {
						return fmt.Errorf("failed to init migrations: %w", err)
					}

					return mg.Drop(ctx)
				},
			},
			{
				Name:  "version",
				Usage: "print current database schema version",
				Flags: []cli.Flag{cfgPathsFlag()},
				Action: func(ctx context.Context, cl *cli.Command) error {
					conf, err := config.Load[migrateConfig](cl.StringSlice("configs"), envPrefix)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					l := logger.NewExtended(append(defaultLoggerOpts(), logger.WithConfig(conf.Log))...)

					mg, err := initMigrations(l, conf.Postgres, true)
					if err != nil {
						return fmt.Errorf("failed to init migrations: %w", err)
					}

					ver, isDirty, err := mg.Version(ctx)
					if err != nil {
						return fmt.Errorf("failed to get version: %w", err)
					}

					l.Infof("version: %d, dirty: %t", ver, isDirty)

					return nil
				},
			},
		},
	}
}
