package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dv-net/dv-processing/internal/blockchains"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/rmanager"
	"github.com/dv-net/dv-processing/internal/services/system"

	"github.com/dv-net/dv-processing/internal/config"
	watcherinterceptors "github.com/dv-net/dv-processing/pkg/dv_watcher/interceptors"
	"github.com/dv-net/dv-proto/gen/go/watcher/addresses/v1/addressesv1connect"
	"github.com/dv-net/dv-proto/gen/go/watcher/subscriber/v1/subscriberv1connect"
	"github.com/dv-net/mx/clients/connectrpc_client"

	"connectrpc.com/connect"
	"github.com/dv-net/dv-processing/api/processing/client/v1/clientv1connect"
	"github.com/dv-net/dv-processing/api/processing/owner/v1/ownerv1connect"
	"github.com/dv-net/dv-processing/api/processing/system/v1/systemv1connect"
	"github.com/dv-net/dv-processing/api/processing/transfer/v1/transferv1connect"
	"github.com/dv-net/dv-processing/api/processing/wallet/v1/walletv1connect"
	"github.com/dv-net/dv-processing/internal/dispatcher"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/escanner"
	"github.com/dv-net/dv-processing/internal/handler"
	"github.com/dv-net/dv-processing/internal/interceptors"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/internal/tscanner"
	"github.com/dv-net/dv-processing/internal/watcher"
	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/mx/launcher"
	"github.com/dv-net/mx/logger"
	"github.com/dv-net/mx/service"
	"github.com/dv-net/mx/service/pingpong"
	"github.com/dv-net/mx/transport/connectrpc_transport"
	"github.com/urfave/cli/v3"
	"go.akshayshah.org/connectproto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/protojson"
)

func startCMD() *cli.Command { //nolint:funlen
	return &cli.Command{
		Name:  "start",
		Usage: "start the server",
		Flags: []cli.Flag{cfgPathsFlag()},
		Action: func(ctx context.Context, cl *cli.Command) error {
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

			// init system service
			systemSvc := system.New(st, version, commitHash)
			pID, err := systemSvc.ProcessingID(ctx)
			if err != nil {
				return fmt.Errorf("processing ID: %w", err)
			}

			appCtx := context.WithValue(ctx, constants.ProcessingIDParamName, pID)
			appCtx = context.WithValue(appCtx, constants.ProcessingVersionParamName, systemSvc.SystemVersion(ctx))

			// init launcher
			ln := launcher.New(
				launcher.WithContext(appCtx),
				launcher.WithVersion(getVersion()),
				launcher.WithName(appName),
				launcher.WithLogger(l),
				launcher.WithRunnerServicesSequence(launcher.RunnerServicesSequenceFifo),
				launcher.WithOpsConfig(conf.Ops),
				launcher.WithAppStartStopLog(true),
			)

			// init explorer proxy service
			explorerProxySvc, err := eproxy.New(appCtx, conf.ExplorerProxy)
			if err != nil {
				return fmt.Errorf("failed to init eproxy service: %w", err)
			}

			var rmanagerSvc *rmanager.Service
			if conf.ResourceManager.Enabled {
				rmanagerSvc, err = rmanager.New(appCtx, l, conf)
				if err != nil {
					return fmt.Errorf("init resource manager service: %w", err)
				}
			}

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

			// init blockchains
			bc, err := blockchains.New(appCtx, conf.Blockchain, sdk)
			if err != nil {
				return fmt.Errorf("failed to init blockchains: %w", err)
			}

			// init dispatchers
			evDispatcher := dispatcher.New()

			// init base services
			baseSvc, err := baseservices.New(appCtx, l, conf, st, explorerProxySvc, bc, systemSvc, evDispatcher, rmanagerSvc, sdk)
			if err != nil {
				return fmt.Errorf("failed to init base services: %w", err)
			}

			// check db version
			if err := baseSvc.System().CheckDBVersion(ctx); err != nil {
				return fmt.Errorf("check db version error: %w", err)
			}

			// init task manager
			tm, err := taskmanager.New(l, conf, st, baseSvc)
			if err != nil {
				return fmt.Errorf("init task manager service: %w", err)
			}

			// init explorer esc
			esc := escanner.New(l, conf.Blockchain, st, baseSvc, tm, baseSvc.Wallets().SDK())

			// init transfer scanner
			tsc := tscanner.New(l, st, baseSvc, tm)

			// init handler
			hdlr := handler.New(l, baseSvc)

			var watcherSvc *watcher.Service
			if conf.Watcher.Enabled {
				watcherSvc, err = initWatcherSvc(ctx, l, conf, st, evDispatcher, baseSvc, pID, version)
				if err != nil {
					return fmt.Errorf("init watcher service: %w", err)
				}
			}

			connectrpcService := connectrpc_transport.NewServer(
				connectrpc_transport.WithLogger(l),
				connectrpc_transport.WithConfig(conf.Grpc),
				connectrpc_transport.WithServices(hdlr.AllServers()...),
				connectrpc_transport.WithServerHandlerWrapper(
					func(h http.Handler) http.Handler {
						return h2c.NewHandler(
							handler.WithCORS(h),
							&http2.Server{})
					},
				),
				connectrpc_transport.WithConnectRPCOptions(
					connect.WithInterceptors(
						interceptors.NewSignInterceptor(
							baseSvc.Clients(),
							conf.Interceptors.DisableCheckingSign,
						),
					),
					connect.WithHandlerOptions(
						connectproto.WithJSON(
							protojson.MarshalOptions{UseProtoNames: true},
							protojson.UnmarshalOptions{DiscardUnknown: true},
						),
					),
				),
				connectrpc_transport.WithReflection(
					clientv1connect.ClientServiceName,
					ownerv1connect.OwnerServiceName,
					transferv1connect.TransferServiceName,
					walletv1connect.WalletServiceName,
					systemv1connect.SystemServiceName,
				),
			)

			// register services
			ln.ServicesRunner().Register(
				service.New(service.WithService(pingpong.New(l))),
				service.New(service.WithService(connectrpcService)),
				service.New(service.WithService(st.Cache())),
				service.New(service.WithService(baseSvc.Webhooks().WebhookServer())),
				service.New(service.WithService(baseSvc.Wallets())),
				service.New(service.WithService(tm)),
				service.New(service.WithService(esc)),
				service.New(service.WithService(tsc)),
				service.New(service.WithService(psql)),
				service.New(service.WithService(watcherSvc)),
				service.New(service.WithService(evDispatcher)),
			)

			if bc.Bitcoin != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.Bitcoin)),
				)
			}

			if bc.Litecoin != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.Litecoin)),
				)
			}

			if bc.BitcoinCash != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.BitcoinCash)),
				)
			}

			if bc.Tron != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.Tron)),
				)
			}

			if bc.Ethereum != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.Ethereum)),
				)
			}

			if bc.BinanceSmartChain != nil {
				ln.ServicesRunner().Register(
					service.New(service.WithService(bc.BinanceSmartChain)),
				)
			}

			if rmanagerSvc == nil {
				l.Infof("resource manager is disabled")
			}

			return ln.Run()
		},
	}
}

func initWatcherSvc(
	ctx context.Context,
	l logger.Logger,
	conf *config.Config,
	st store.IStore,
	evDispatcher dispatcher.IService,
	bs baseservices.IBaseServices,
	processingID, version string,
) (*watcher.Service, error) {
	connectOpts := connectrpc_client.WithConnectrpcOpts(
		connect.WithGRPC(),
		connect.WithInterceptors(
			interceptors.NewProcessingIdentity(processingID, version),
			watcherinterceptors.NewWatcherAuthInterceptor(
				processingID,
				conf.Watcher.ClientSecret,
			),
		),
	)

	subscriberCl, err := connectrpc_client.New(
		conf.Watcher.Connect, l, subscriberv1connect.NewSubscriberServiceClient,
		connectrpc_client.WithName("watcher-clients-client"),
		connectrpc_client.WithContext(ctx),
		connectOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("watcher clients init: %w", err)
	}

	addrCl, err := connectrpc_client.New(
		conf.Watcher.Connect, l, addressesv1connect.NewAddressesServiceClient,
		connectrpc_client.WithName("watcher-addresses-client"),
		connectrpc_client.WithContext(ctx),
		connectOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("watcher addresses init: %w", err)
	}

	return watcher.New(
		l,
		conf.Watcher,
		st,
		conf.Blockchain,
		subscriberCl,
		addrCl,
		bs,
		evDispatcher,
	), nil
}
