package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dv-net/dv-processing/internal/eproxy"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
	addressesv1 "github.com/dv-net/dv-proto/gen/go/watcher/addresses/v1"
	"github.com/dv-net/dv-proto/gen/go/watcher/addresses/v1/addressesv1connect"
	subscriberv1 "github.com/dv-net/dv-proto/gen/go/watcher/subscriber/v1"
	"github.com/dv-net/dv-proto/gen/go/watcher/subscriber/v1/subscriberv1connect"

	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/dispatcher"

	"connectrpc.com/connect"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
)

const serviceName = "watcher"

type Service struct {
	recTime time.Duration

	addr  addressesv1connect.AddressesServiceClient
	cl    subscriberv1connect.SubscriberServiceClient
	log   logger.Logger
	bs    baseservices.IBaseServices
	store store.IStore

	blockchains      config.Blockchain
	walletSubscriber dispatcher.IService
}

func New(
	l logger.Logger,
	conf config.Watcher,
	st store.IStore,
	blockchains config.Blockchain,
	client subscriberv1connect.SubscriberServiceClient,
	addr addressesv1connect.AddressesServiceClient,
	bs baseservices.IBaseServices,
	walletSubscriber dispatcher.IService,
) *Service {
	svc := &Service{
		blockchains:      blockchains,
		cl:               client,
		addr:             addr,
		log:              l,
		bs:               bs,
		store:            st,
		walletSubscriber: walletSubscriber,

		recTime: conf.GrpcReconnectionDelay,
	}

	return svc
}

func (s *Service) Name() string {
	return serviceName
}

func (s *Service) Stop(_ context.Context) error {
	return nil
}

func (s *Service) Ping(_ context.Context) error { return nil }

func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return nil
	}

	// handle target blockchains
	eg, egCtx := errgroup.WithContext(ctx)
	for _, blockchain := range s.blockchains.Available() {
		if handler := s.handlerByBlockchain(blockchain); handler != nil {
			s.log.Debugw("mempool watcher started", "blockchain", blockchain)

			eg.Go(func() error {
				return retry.New(
					retry.WithPolicy(retry.PolicyInfinite),
					retry.WithDelay(s.recTime),
					retry.WithContext(egCtx),
				).Do(func() error {
					return handler(egCtx)
				})
			})
		}
	}

	// init base
	err := retry.New(
		retry.WithPolicy(retry.PolicyInfinite),
		retry.WithContext(egCtx),
	).Do(func() error {
		return s.initWorkers(egCtx)
	})
	if err != nil {
		return fmt.Errorf("init watcher service: %w", err)
	}

	return eg.Wait()
}

func (s *Service) initWorkers(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	// initialize target wallets list in watcher
	eg.Go(func() error {
		return s.initWalletsList(egCtx)
	})

	// append all new created wallets to watchlist
	eg.Go(func() error {
		if err := s.processNewHotWallets(egCtx); err != nil {
			return fmt.Errorf("watcher process new hot wallets: %w", err)
		}
		return nil
	})

	if errFromGroup := eg.Wait(); errFromGroup != nil {
		return errFromGroup
	}

	<-ctx.Done()

	return nil
}

func (s *Service) handlerByBlockchain(blockchain wconstants.BlockchainType) func(context.Context) error {
	if !blockchain.IsBitcoinLike() {
		// Only btc-like mempool subscriber available
		return nil
	}

	return func(ctx context.Context) error {
		stream, err := s.cl.SubscribeMempool(
			ctx,
			connect.NewRequest(&subscriberv1.SubscribeMempoolRequest{
				Blockchain: eproxy.ConvertBlockchain(blockchain),
			}),
		)
		if err != nil {
			return fmt.Errorf("prepare stream err: %w", err)
		}

		errCh := make(chan error, 1)
		if stream != nil {
			go func() {
				errCh <- s.processMempoolStream(ctx, stream, blockchain)
			}()
		}

		select {
		case streamErr := <-errCh:
			return fmt.Errorf("watcher strean: %w", streamErr)
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Service) initWalletsList(ctx context.Context) error {
	hotWallets, err := s.bs.Wallets().Hot().GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get hot wallets for watcher:%w", err)
	}

	err = s.UpdateAddressesList(ctx, hotWallets)
	if err != nil {
		return fmt.Errorf("update addresses list: %w", err)
	}

	return nil
}

func (s *Service) processNewHotWallets(ctx context.Context) error {
	hotWalletsCh := s.walletSubscriber.CreatedHotWalletDispatcher().Subscribe()
	defer s.walletSubscriber.CreatedHotWalletDispatcher().Unsubscribe(hotWalletsCh)

	for {
		select {
		case newHotWallet, ok := <-hotWalletsCh:
			if !ok {
				continue
			}
			preparedAddr, err := s.convertHotWalletToPb(newHotWallet)
			if err != nil {
				s.log.Errorw("convert hot wallet to pb", "error", err)
				continue
			}
			_, err = s.addr.AppendAddressesToWatchList(ctx, connect.NewRequest(
				&addressesv1.AppendAddressesToWatchListRequest{
					Addresses: []*addressesv1.Address{preparedAddr},
				},
			))
			if err != nil {
				s.log.Errorw("append address to watchlist", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Service) UpdateAddressesList(ctx context.Context, wallets []*models.HotWallet) error {
	if err := retry.New().Do(func() error {
		_, err := s.addr.UpdateWatchList(ctx, connect.NewRequest(&addressesv1.UpdateWatchListRequest{
			Addresses: s.convertHotWalletsToPb(wallets),
		}))
		if err != nil {
			return fmt.Errorf("update addresses list: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update addresses list: %w", err)
	}

	return nil
}

func (s *Service) processMempoolTxEvents(
	ctx context.Context,
	blockchain wconstants.BlockchainType,
	tx *transactionsv2.Transaction,
) error {
	err := pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		batchCreateParams := make([]webhooks.BatchCreateParams, 0, len(tx.Events))
		for _, event := range tx.Events {
			if event.AddressTo == nil || *event.AddressTo == "" {
				continue
			}

			// check if the address belongs to us
			checkResult, err := s.bs.Wallets().CheckWallet(ctx, blockchain, *event.AddressTo)
			if err != nil && !errors.Is(err, wallets.ErrAddressNotFound) {
				return fmt.Errorf("check wallet from: %w", err)
			}
			if errors.Is(err, wallets.ErrAddressNotFound) {
				s.log.Error("irrelevant tx from watcher", err)
				continue
			}

			transactionData := webhooks.TransactionData{
				Hash:          tx.Hash,
				Confirmations: tx.Confirmations,
				Fee:           &tx.Fee,
			}

			if tx.CreatedAt != nil {
				transactionData.CreatedAt = tx.CreatedAt.AsTime()
			}

			transactionEventData := webhooks.TransactionEventData{
				AddressFrom:       event.AddressFrom,
				AddressTo:         event.AddressTo,
				Value:             event.Value,
				AssetIdentify:     event.AssetIdentifier,
				BlockchainUniqKey: event.BlockchainUniqKey,
			}

			whCreateParamsData := webhooks.EventTransactionCreateParamsData{
				Blockchain:       blockchain,
				Tx:               transactionData,
				Event:            transactionEventData,
				WebhookKind:      models.WebhookKindDeposit,
				WalletType:       checkResult.WalletType,
				WebhookStatus:    models.WebhookEventStatusInMempool,
				OwnerID:          checkResult.OwnerID,
				ExternalWalletID: checkResult.ExternalWalletID,
			}

			params, err := s.bs.Webhooks().EventTransactionCreateParams(ctx, whCreateParamsData)
			if err != nil {
				return fmt.Errorf("prepare wh params for mempool tx: %w", err)
			}
			batchCreateParams = append(batchCreateParams, params)
		}

		if err := s.bs.Webhooks().BatchCreate(ctx, batchCreateParams, repos.WithTx(dbTx)); err != nil {
			return fmt.Errorf("batch create webhooks by mempool tx event: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("process mempool tx events failed: %w", err)
	}

	return nil
}
