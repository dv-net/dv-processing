package fsmtron_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/internal/blockchains"
	"github.com/dv-net/dv-processing/internal/services/system"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/dispatcher"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/fsm/fsmtron"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/dv-processing/pkg/testutils"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/mx/cfg"
	"github.com/dv-net/mx/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func initLogger() logger.ExtendedLogger {
	return logger.NewExtended(
		logger.WithLogFormat(logger.LoggerFormatConsole),
		logger.WithLogLevel(logger.LogLevelDebug),
	)
}

func initConfig() (*config.Config, error) {
	conf := new(config.Config)
	if err := cfg.Load(conf,
		cfg.WithLoaderConfig(cfg.Config{
			Files:      []string{"../../../../config.yaml"},
			SkipFlags:  true,
			MergeFiles: true,
		}),
	); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return conf, nil
}

func initStore(l logger.Logger, conf *config.Config) (store.IStore, error) {
	// init postgres connection
	psql, err := postgres.New(context.Background(), conf.Postgres, l)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	// init store
	st := store.New(psql)

	return st, nil
}

func TestFSM(t *testing.T) {
	l := initLogger()
	conf, err := initConfig()
	require.NoError(t, err)

	st, err := initStore(l, conf)
	require.NoError(t, err)

	appCtx := testutils.GetContext()

	explorerProxySvc, err := eproxy.New(appCtx, conf.ExplorerProxy)
	require.NoError(t, err)

	// init wallet sdk
	sdk := walletsdk.New(walletsdk.Config{
		Bitcoin: walletsdk.BitcoinConfig{
			Network: conf.Blockchain.Bitcoin.Network,
		},
		Litecoin: walletsdk.LitecoinConfig{
			Network: conf.Blockchain.Litecoin.Network,
		},
		BitcoinCash: walletsdk.BitcoinCashConfig{
			Network: conf.Blockchain.BitcoinCash.Network,
		},
	})

	bc, err := blockchains.New(context.Background(), conf.Blockchain, sdk)
	require.NoError(t, err)

	sysSvc := system.New(l, st, "sysVersion", "sysCommit")
	bs, err := baseservices.New(appCtx, l, conf, st, explorerProxySvc, bc, sysSvc, dispatcher.New(), nil, sdk)
	require.NoError(t, err)

	transfer, err := bs.Transfers().GetByID(appCtx, uuid.MustParse("cdb8670e-c0ae-49af-8e1a-9f866026dd19"))
	require.NoError(t, err)

	// create a new TronFSM
	fsm, err := fsmtron.NewFSM(l, conf, st, bs, transfer)
	require.NoError(t, err)

	// run the workflow
	err = fsm.Run(appCtx)
	require.NoError(t, err)
}

func TestReadFromCh(t *testing.T) {
	ch := make(chan int)

	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()

	for item := range ch {
		fmt.Println(item)
	}

	fmt.Println("done")
}
