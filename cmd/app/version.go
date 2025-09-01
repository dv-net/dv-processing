package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func versionCMD() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the version",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println(appName + " version" + getBuildInfo())
			return nil
		},
	}
}
