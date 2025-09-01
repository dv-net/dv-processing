package main

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/blockchains"
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/dispatcher"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/system"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/encryption"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/logger"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/urfave/cli/v3"
)

func blockchainCMD() *cli.Command {
	return &cli.Command{
		Name:  "blockchain",
		Usage: "blockchain tools",
		Commands: []*cli.Command{
			tronCMD(),
		},
	}
}

func tronCMD() *cli.Command {
	return &cli.Command{
		Name:  "tron",
		Usage: "tron tools",
		Commands: []*cli.Command{
			tronReclaimResourceCMD(),
		},
	}
}

func tronReclaimResourceCMD() *cli.Command {
	return &cli.Command{
		Name:  "reclaim-resource",
		Usage: "reclaim resource",
		Flags: []cli.Flag{
			cfgPathsFlag(),
			&cli.StringFlag{
				Name:     "processing-address",
				Aliases:  []string{"pa"},
				Required: true,
				Usage:    "processing wallet address",
			},
			&cli.StringFlag{
				Name:     "destination-address",
				Aliases:  []string{"da"},
				Required: true,
				Usage:    "destination address",
			},
			&cli.StringFlag{
				Name:     "type",
				Required: true,
				Usage:    "resource type (bandwidth or energy)",
			},
		},
		Action: func(ctx context.Context, cl *cli.Command) error {
			conf, err := config.Load[config.Config](cl.StringSlice("configs"), envPrefix)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			loggerOpts := append(defaultLoggerOpts(), logger.WithConfig(conf.Log))

			l := logger.NewExtended(loggerOpts...)
			defer func() {
				_ = l.Sync()
			}()

			if !conf.Blockchain.Tron.Enabled {
				return fmt.Errorf("tron is disabled")
			}

			// init postgres connection
			psql, err := postgres.New(ctx, conf.Postgres, l)
			if err != nil {
				return fmt.Errorf("failed to init postgres: %w", err)
			}

			// init store
			st := store.New(psql)

			// init system service
			systemSvc := system.New(st, version, commitHash)
			pID, err := systemSvc.ProcessingID(ctx)
			if err != nil {
				return fmt.Errorf("processing ID: %w", err)
			}

			appCtx := context.WithValue(ctx, constants.ProcessingIDParamName, pID)
			appCtx = context.WithValue(appCtx, constants.ProcessingVersionParamName, systemSvc.SystemVersion(ctx))

			// init explorer proxy service
			explorerProxySvc, err := eproxy.New(appCtx, conf.ExplorerProxy)
			if err != nil {
				return fmt.Errorf("failed to init eproxy service: %w", err)
			}

			identity, err := constants.IdentityFromContext(appCtx)
			if err != nil {
				return fmt.Errorf("get processing identity: %w", err)
			}

			// init tron service and make it unnecessary if tron is disabled
			tronService, err := tron.NewTron(conf.Blockchain.Tron.ConvertToSDKConfig(identity))
			if err != nil {
				return fmt.Errorf("failed to init tron service: %w", err)
			}

			if err := tronService.Start(appCtx); err != nil {
				return fmt.Errorf("failed to start tron service: %w", err)
			}

			defer func() {
				_ = tronService.Stop(appCtx)
			}()

			// init wallet sdk
			sdk := walletsdk.New(walletsdk.Config{})

			bc, err := blockchains.New(appCtx, conf.Blockchain, sdk)
			if err != nil {
				return fmt.Errorf("failed to init blockchains: %w", err)
			}

			// init base services
			dsptch := dispatcher.New()
			baseSvc, err := baseservices.New(appCtx, l, conf, st, explorerProxySvc, bc, systemSvc, dsptch, nil, sdk)
			if err != nil {
				return fmt.Errorf("failed to init base services: %w", err)
			}

			// get processing wallet
			processingWallet, err := baseSvc.Wallets().Processing().Get(appCtx, wconstants.BlockchainTypeTron, cl.String("processing-address"))
			if err != nil {
				return fmt.Errorf("get processing wallet: %w", err)
			}

			owner, err := baseSvc.Owners().GetByID(appCtx, processingWallet.OwnerID)
			if err != nil {
				return fmt.Errorf("get owner: %w", err)
			}

			var resourceType core.ResourceCode
			switch cl.String("type") {
			case "bandwidth":
				resourceType = core.ResourceCode_BANDWIDTH
			case "energy":
				resourceType = core.ResourceCode_ENERGY
			default:
				return fmt.Errorf("invalid resource type %s", cl.String("type"))
			}

			mnemonic := owner.Mnemonic
			if conf.IsEnabledSeedEncryption() {
				mnemonic, err = encryption.Decrypt(mnemonic, owner.ID.String())
				if err != nil {
					return fmt.Errorf("decrypt mnemonic: %w", err)
				}
			}

			reclaimResourceParams := tron.ReclaimResourceParams{
				FromAddress:  processingWallet.Address,
				FromSequence: uint32(processingWallet.Sequence), //nolint:gosec
				Mnemonic:     mnemonic,
				PassPhrase:   owner.PassPhrase.String,
				ToAddress:    cl.String("destination-address"),
				ResourceType: resourceType,
			}

			l.Infow(
				"reclaim resource",
				"from", reclaimResourceParams.FromAddress,
				"to", reclaimResourceParams.ToAddress,
				"resource", cl.String("type"),
			)

			tx, err := tronService.ReclaimResource(appCtx, reclaimResourceParams)
			if err != nil {
				return fmt.Errorf("reclaim resource: %w", err)
			}

			l.Infof("tx hash: %s", common.Bytes2Hex(tx.Txid))

			return nil
		},
	}
}
