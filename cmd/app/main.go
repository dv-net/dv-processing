package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/mx/logger"
	mxsignal "github.com/dv-net/mx/util/signal"
	"github.com/urfave/cli/v3"
)

var (
	appName             = "dv-processing"
	version             = "local"
	commitHash          = "unknown"
	buildDate           = "unknown"
	envPrefix           = "PROCESSING"
	logMemoryBufferSize = 1000
)

func getVersion() string { return version + "-" + commitHash }

func getBuildInfo() string {
	return fmt.Sprintf(
		"\nrelease: %s\ncommit hash: %s\nbuild date: %s\ngo version: %s",
		version,
		commitHash,
		buildDate,
		runtime.Version(),
	)
}

func defaultLoggerOpts() []logger.Option {
	return []logger.Option{
		logger.WithAppName(appName),
		logger.WithAppVersion(getVersion()),
		logger.WithMemoryBuffer(logMemoryBufferSize),
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), mxsignal.Shutdown()...)
	defer cancel()

	// init default logger
	l := logger.NewExtended(defaultLoggerOpts()...)

	// try to load config
	if conf, err := config.Load[config.Config]([]string{"config.yaml"}, envPrefix); err == nil {
		l = logger.NewExtended(append(defaultLoggerOpts(), logger.WithConfig(conf.Log))...)
	}

	app := &cli.Command{
		Name:    appName,
		Version: getVersion(),
		Suggest: true,
		Commands: []*cli.Command{
			configCMD(),
			startCMD(),
			migrateCMD(),
			blockchainCMD(),
			utilsCMD(),
			versionCMD(),
		},
	}

	// run cli runner
	if err := app.Run(ctx, os.Args); err != nil {
		l.Fatalf("failed to run cli runner: %s", err)
	}
}
