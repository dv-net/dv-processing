package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/xconfig"

	"github.com/goccy/go-yaml"
	"github.com/urfave/cli/v3"
)

func cfgPathsFlag() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:    "configs",
		Aliases: []string{"c"},
		Usage:   "allows you to use your own paths to configuration files, separated by commas (config.yaml,config.prod.yml,.env)",
		Value:   cli.NewStringSlice("config.yaml").Value(),
	}
}

func configCMD() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "validate, gen envs and flags for config",
		Commands: []*cli.Command{
			{
				Name:  "genenvs",
				Usage: "generate markdown for all envs and config yaml template",
				Action: func(_ context.Context, _ *cli.Command) error {
					conf := new(config.Config)

					_, err := xconfig.Load(conf, xconfig.WithEnvPrefix(envPrefix))
					if err != nil {
						return fmt.Errorf("failed to generate markdown: %w", err)
					}

					buf := bytes.NewBuffer(nil)
					enc := yaml.NewEncoder(buf, yaml.Indent(2))
					defer enc.Close()

					if err := enc.Encode(conf); err != nil {
						return fmt.Errorf("failed to encode yaml: %w", err)
					}

					if err := os.WriteFile("config.template.yaml", buf.Bytes(), 0o600); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}

					return nil
				},
			},
			{
				Name:  "validate",
				Usage: "validate config without starting the server",
				Flags: []cli.Flag{cfgPathsFlag()},
				Action: func(_ context.Context, cl *cli.Command) error {
					_, err := config.Load[config.Config](cl.StringSlice("configs"), envPrefix)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "usage",
				Usage: "print config usage without starting the server",
				Action: func(_ context.Context, _ *cli.Command) error {
					conf := new(config.Config)
					c, err := xconfig.Load(conf, xconfig.WithEnvPrefix(envPrefix))
					if err != nil {
						return err
					}
					usage, err := c.Usage()
					if err != nil {
						return err
					}
					fmt.Println(usage)
					return nil
				},
			},
		},
	}
}
