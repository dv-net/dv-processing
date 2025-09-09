package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	processing "github.com/dv-net/dv-processing"
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/services/owners"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/mx/logger"
	"github.com/dv-net/xconfig"
	"github.com/urfave/cli/v3"
)

func utilsCMD() *cli.Command {
	return &cli.Command{
		Name:  "utils",
		Usage: "custom cli utils",
		Commands: []*cli.Command{
			genReadmeCMD(),
			compressSeedsCMD(),
			compressOTPDataCMD(),
		},
	}
}

func genReadmeCMD() *cli.Command {
	return &cli.Command{
		Name:  "readme",
		Usage: "generate markdown for all envs and config yaml template",
		Action: func(_ context.Context, cl *cli.Command) error {
			envMarkdown, err := xconfig.GenerateMarkdown(new(config.Config), xconfig.WithEnvPrefix(envPrefix))
			if err != nil {
				return fmt.Errorf("failed to generate markdown: %w", err)
			}

			output := new(bytes.Buffer)
			cl.Root().Writer = output
			if err := cli.ShowAppHelp(cl.Root()); err != nil {
				return err
			}

			tmpl, err := template.ParseFS(processing.ReadmeFS, "README.go.tmpl")
			if err != nil {
				return err
			}

			data := struct {
				AppName      string
				AppBin       string
				AppUsage     string
				AppHelp      string
				Environments string
			}{
				AppName:      strings.ReplaceAll(strings.ToTitle(appName), "-", " "),
				AppBin:       strings.ReplaceAll(appName, "-", "_"),
				AppUsage:     cl.Usage,
				AppHelp:      output.String(),
				Environments: envMarkdown,
			}

			buf := bytes.NewBuffer(nil)
			if err := tmpl.ExecuteTemplate(buf, "readme", data); err != nil {
				return err
			}

			if err := os.WriteFile("README.md", buf.Bytes(), 0o600); err != nil {
				return err
			}

			return nil
		},
	}
}

func compressSeedsCMD() *cli.Command { //nolint:dupl
	return &cli.Command{
		Name:  "seeds",
		Usage: "encrypt and decrypt seeds for all owners",
		Flags: []cli.Flag{
			cfgPathsFlag(),
			&cli.BoolFlag{
				Name:  "encrypt",
				Usage: "encrypt seeds",
			},
			&cli.BoolFlag{
				Name:  "decrypt",
				Usage: "decrypt seeds",
			},
		},
		Action: func(ctx context.Context, cl *cli.Command) error {
			if cl.Bool("encrypt") && cl.Bool("decrypt") {
				return fmt.Errorf("only one of encrypt or decrypt can be set")
			}

			if !cl.Bool("encrypt") && !cl.Bool("decrypt") {
				return fmt.Errorf("one of encrypt or decrypt must be set")
			}

			conf, err := config.Load[config.Config](cl.StringSlice("configs"), envPrefix)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			loggerOpts := append(defaultLoggerOpts(), logger.WithConfig(conf.Log))

			l := logger.NewExtended(loggerOpts...)
			defer func() { _ = l.Sync() }()

			// init postgres connection
			psql, err := postgres.New(ctx, conf.Postgres, l)
			if err != nil {
				return fmt.Errorf("failed to init postgres: %w", err)
			}

			// init store
			st := store.New(psql)

			ownersService := owners.New(conf, st, nil)

			if cl.Bool("encrypt") {
				if err := ownersService.EncryptSeedsForAllOwners(ctx); err != nil {
					return fmt.Errorf("failed to encrypt seeds: %w", err)
				}
			}

			if cl.Bool("decrypt") {
				if err := ownersService.DecryptSeedsForAllOwners(ctx); err != nil {
					return fmt.Errorf("failed to decrypt seeds: %w", err)
				}
			}

			return nil
		},
	}
}

func compressOTPDataCMD() *cli.Command { //nolint:dupl
	return &cli.Command{
		Name:  "otp-data",
		Usage: "encrypt and decrypt OTP data for all owners",
		Flags: []cli.Flag{
			cfgPathsFlag(),
			&cli.BoolFlag{
				Name:  "encrypt",
				Usage: "encrypt OTP data",
			},
			&cli.BoolFlag{
				Name:  "decrypt",
				Usage: "decrypt OTP data",
			},
		},
		Action: func(ctx context.Context, cl *cli.Command) error {
			if cl.Bool("encrypt") && cl.Bool("decrypt") {
				return fmt.Errorf("only one of encrypt or decrypt can be set")
			}

			if !cl.Bool("encrypt") && !cl.Bool("decrypt") {
				return fmt.Errorf("one of encrypt or decrypt must be set")
			}

			conf, err := config.Load[config.Config](cl.StringSlice("configs"), envPrefix)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			loggerOpts := append(defaultLoggerOpts(), logger.WithConfig(conf.Log))

			l := logger.NewExtended(loggerOpts...)
			defer func() { _ = l.Sync() }()

			// init postgres connection
			psql, err := postgres.New(ctx, conf.Postgres, l)
			if err != nil {
				return fmt.Errorf("failed to init postgres: %w", err)
			}

			// init store
			st := store.New(psql)

			ownersService := owners.New(conf, st, nil)

			if cl.Bool("encrypt") {
				if err := ownersService.EncryptOTPDataForAllOwners(ctx); err != nil {
					return fmt.Errorf("failed to encrypt OTP data: %w", err)
				}
			}

			if cl.Bool("decrypt") {
				if err := ownersService.DecryptOTPDataForAllOwners(ctx); err != nil {
					return fmt.Errorf("failed to decrypt OTP data: %w", err)
				}
			}

			return nil
		},
	}
}
