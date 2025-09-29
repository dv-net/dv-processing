package escanner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfer_transactions"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
)

type scanner struct {
	logger     logger.Logger
	conf       config.Blockchain
	blockchain wconstants.BlockchainType
	store      store.IStore
	bs         baseservices.IBaseServices
	tm         *taskmanager.TaskManager
	sdk        *walletsdk.SDK

	lastNodeBlockHeight   atomic.Int64
	lastParsedBlockHeight atomic.Int64
}

func newScanner(
	l logger.Logger,
	c config.Blockchain,
	st store.IStore,
	bs baseservices.IBaseServices,
	tm *taskmanager.TaskManager,
	sdk *walletsdk.SDK,
	blockchain wconstants.BlockchainType,
) *scanner {
	return &scanner{
		logger: logger.With(l,
			"module", "scanner",
			"blockchain", blockchain.String(),
		),
		conf:       c,
		store:      st,
		bs:         bs,
		tm:         tm,
		sdk:        sdk,
		blockchain: blockchain,
	}
}

// start
func (s *scanner) start(ctx context.Context) error {
	// get last block from the database
	lastDBBlockNumber, err := s.bs.ProcessedBlocks().LastBlockNumber(ctx, s.blockchain)
	if err != nil {
		if !errors.Is(err, storecmn.ErrNotFound) {
			return err
		}

		s.lastParsedBlockHeight.Store(-1)
	} else {
		s.lastParsedBlockHeight.Store(lastDBBlockNumber)
	}

	// start loop for getting new last block from the proxy explorer
	go func() {
		for ctx.Err() == nil {
			fn := func() error {
				lastNodeBlockHeight, err := s.bs.EProxy().LastBlockNumber(ctx, s.blockchain)
				if err != nil {
					return fmt.Errorf("get last block from explorer: %w", err)
				}

				if s.lastNodeBlockHeight.Load() == int64(lastNodeBlockHeight) { //nolint:gosec
					return nil
				}

				s.lastNodeBlockHeight.Store(int64(lastNodeBlockHeight)) //nolint:gosec

				return nil
			}

			if err := fn(); err != nil {
				s.logger.Error(err.Error())
			}

			// TODO: edit it later
			time.Sleep(1 * time.Second)
		}
	}()

	// handle new blocks
	go func() {
		for ctx.Err() == nil {
			if err := s.handleBlocks(ctx); err != nil {
				s.logger.Error(err.Error())

				time.Sleep(1 * time.Second)
			}
		}
	}()

	<-ctx.Done()

	return nil
}

// handleBlock
func (s *scanner) handleBlocks(ctx context.Context) error {
	lpBlock := s.lastParsedBlockHeight.Load()
	lnBlock := s.lastNodeBlockHeight.Load()

	if lpBlock >= lnBlock || lnBlock == 0 {
		time.Sleep(1 * time.Second)
		return nil
	}

	existsLastBlockDB := true
	if lpBlock == -1 {
		lpBlock = lnBlock - 1
		existsLastBlockDB = false
	}

	for i := lpBlock + 1; i <= lnBlock; i++ {
		if ctx.Err() != nil {
			return nil //nolint:nilerr
		}

		if err := s.handleBlock(i, existsLastBlockDB); err != nil { //nolint:contextcheck
			return fmt.Errorf("handle block %d: %w", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	return nil
}

type createWebhookParams struct {
	tx          *transactionsv2.Transaction
	event       *transactionsv2.Event
	whKind      models.WebhookKind
	checkResult wallets.CheckWalletResult
}

func (cwp *createWebhookParams) IsTrxContractActivationDeposit(activationContractAddr string) bool {
	return cwp.checkResult.WalletType == constants.WalletTypeHot &&
		cwp.whKind == models.WebhookKindDeposit && // deposit
		cwp.event != nil &&
		cwp.event.GetAssetIdentifier() != "" &&
		strings.EqualFold(cwp.event.GetAssetIdentifier(), tron.TrxAssetIdentifier) &&
		cwp.event.AddressFrom != nil &&
		*cwp.event.AddressFrom != "" &&
		strings.EqualFold(*cwp.event.AddressFrom, activationContractAddr) &&
		cwp.event.AddressTo != nil &&
		*cwp.event.AddressTo != ""
}

func (cwp *createWebhookParams) IsTrxHotWalletDeposit() bool {
	return cwp.checkResult.WalletType == constants.WalletTypeHot &&
		cwp.whKind == models.WebhookKindDeposit && // deposit
		cwp.event != nil &&
		cwp.event.GetAssetIdentifier() != "" &&
		strings.EqualFold(cwp.event.GetAssetIdentifier(), tron.TrxAssetIdentifier) && // trx asset
		cwp.event.AddressTo != nil &&
		*cwp.event.AddressTo != ""
}

// handleBlock
func (s *scanner) handleBlock(blockHeight int64, existsLastBlockDB bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	s.logger.Debugf("start processing block %d", blockHeight)

	now := time.Now()
	txs, err := s.bs.EProxy().FindTransactions(ctx, s.blockchain, eproxy.FindTransactionsParams{
		BlockHeight: utils.Pointer(uint64(blockHeight)), //nolint:gosec
	})
	if err != nil {
		return fmt.Errorf("find transactions: %w", err)
	}

	s.logger.Debugf("found %d transactions in block %d in %s", len(txs), blockHeight, time.Since(now))

	createWhParams := utils.NewSlice[createWebhookParams]()

	eg, gCtx := errgroup.WithContext(ctx)
	eg.SetLimit(100)

	now = time.Now()
	for _, tx := range txs {
		eg.Go(func() error {
			for _, event := range tx.Events {
				fn := func() error {
					if event.Type == nil || *event.Type != transactionsv2.EventType_EVENT_TYPE_TRANSFER {
						return nil
					}

					if event.Status != nil && *event.Status != transactionsv2.EventStatus_EVENT_STATUS_SUCCESS {
						return nil
					}
					// skip zero transfer if transaction initiator is another address. approve if we send commission
					if event.GetValue() == "0" && event.GetAddressFrom() != "" && event.GetAddressFrom() != tx.GetAddressFrom() {
						return nil
					}

					for _, check := range s.checksForEvent(event) {
						// check if the address belongs to us
						result, err := s.bs.Wallets().CheckWallet(gCtx, s.blockchain, check.address)
						if err != nil {
							// skip if the address does not belong to us
							if errors.Is(err, wallets.ErrAddressNotFound) {
								continue
							}
							return fmt.Errorf("check wallet: %w", err)
						}

						createWhParams.Add(createWebhookParams{
							tx:          tx,
							event:       event,
							whKind:      check.kind,
							checkResult: *result,
						})
					}

					return nil
				}

				if err := fn(); err != nil {
					return fmt.Errorf("check event: %w", err)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("wait for all transactions: %w", err)
	}

	// create webhooks

	// handle block in postgres transaction
	if err := pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if len(createWhParams.GetAll()) != 0 {
			batchParams := make([]webhooks.BatchCreateParams, 0, createWhParams.Len())
			for _, params := range createWhParams.GetAll() {
				transactionData := webhooks.TransactionData{
					Hash:          params.tx.Hash,
					Confirmations: params.tx.Confirmations,
					Fee:           &params.tx.Fee,
				}

				if params.tx.Status != "" {
					transactionData.Status = &params.tx.Status
				}

				if params.tx.CreatedAt != nil {
					transactionData.CreatedAt = params.tx.CreatedAt.AsTime()
				}

				event := params.event
				transactionEventData := webhooks.TransactionEventData{
					AddressFrom:       event.AddressFrom,
					AddressTo:         event.AddressTo,
					Value:             event.Value,
					AssetIdentify:     event.AssetIdentifier,
					BlockchainUniqKey: event.BlockchainUniqKey,
				}

				whCreateParamsData := webhooks.EventTransactionCreateParamsData{
					Blockchain:       s.blockchain,
					Tx:               transactionData,
					Event:            transactionEventData,
					WebhookKind:      params.whKind,
					WalletType:       params.checkResult.WalletType,
					WebhookStatus:    models.WebhookEventStatusWaitingConfirmations,
					OwnerID:          params.checkResult.OwnerID,
					ExternalWalletID: params.checkResult.ExternalWalletID,
				}

				// post create webhook handler
				if err := s.beforeCreateWhHandler(ctx, dbTx, params, &whCreateParamsData); err != nil {
					return fmt.Errorf("post create webhook handler: %w", err)
				}

				param, err := s.bs.Webhooks().EventTransactionCreateParams(ctx, whCreateParamsData)
				if err != nil {
					return fmt.Errorf("get params for webhook: %w", err)
				}

				batchParams = append(batchParams, param)

				var address string
				if params.whKind == models.WebhookKindTransfer && params.event.AddressFrom != nil {
					address = *params.event.AddressFrom
				} else if params.whKind == models.WebhookKindDeposit && params.event.AddressTo != nil {
					address = *params.event.AddressTo
				}

				var blockchainUniqKey string
				if params.event.BlockchainUniqKey != nil {
					blockchainUniqKey = *params.event.BlockchainUniqKey
				}

				_, err = s.tm.Client().InsertTx(ctx, dbTx,
					taskmanager.WebhookWaitingConfirmationsArgs{
						Blockchain:             s.blockchain,
						Hash:                   params.tx.Hash,
						Address:                address,
						EventBlockchainUniqKey: blockchainUniqKey,
						WebhookKind:            params.whKind,
						WalletType:             params.checkResult.WalletType,
						OwnerID:                params.checkResult.OwnerID,
						ExternalWalletID:       params.checkResult.ExternalWalletID,
						IsSystem:               whCreateParamsData.IsSystem,
					},
					&river.InsertOpts{
						UniqueOpts: river.UniqueOpts{
							ByArgs: true,
						},
						ScheduledAt: time.Now().Add(constants.ConfirmationsTimeout(s.blockchain, params.tx.Confirmations)),
					},
				)
				if err != nil {
					return fmt.Errorf("insert webhook job: %w", err)
				}
			}

			if err := s.bs.Webhooks().BatchCreate(ctx, batchParams, repos.WithTx(dbTx)); err != nil {
				return err
			}
		}

		s.logger.Debugf("processed block %d in %s", blockHeight, time.Since(now))

		if existsLastBlockDB {
			if err := s.bs.ProcessedBlocks().UpdateNumber(ctx, s.blockchain, blockHeight, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("update block number: %w", err)
			}

			s.logger.Debugf("updated last block from explorer %s is %d", s.blockchain.String(), blockHeight)
		} else {
			if err := s.bs.ProcessedBlocks().Create(ctx, s.blockchain, blockHeight, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("create block number: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	s.lastParsedBlockHeight.Store(blockHeight)

	return nil
}

// beforeCreateWhHandler - before create webhook handler
func (s *scanner) beforeCreateWhHandler(
	ctx context.Context,
	dbTx pgx.Tx,
	params createWebhookParams,
	whCreateParamsData *webhooks.EventTransactionCreateParamsData,
) error {
	if !s.blockchain.IsSystemTransactionsSupported() {
		return nil
	}

	// Fetch system transactions with existing transfers by tx_hash
	systemTxs, err := s.store.TransferTransactions(repos.WithTx(dbTx)).FindSystemTransactions(ctx, repo_transfer_transactions.FindSystemTransactionsParams{
		OwnerID:       whCreateParamsData.OwnerID,
		Blockchain:    s.blockchain,
		TxHash:        params.tx.Hash,
		SystemTxTypes: models.TransferTransactionSystemTypes(),
	})
	if err != nil {
		return fmt.Errorf("find system transactions: %w", err)
	}

	// transaction occurred as system
	if len(systemTxs) > 0 {
		whCreateParamsData.IsSystem = true
	}

	if s.blockchain == wconstants.BlockchainTypeTron && !params.checkResult.Activated() {
		// Check if wallet activation is required
		if params.IsTrxContractActivationDeposit(s.conf.Tron.ActivationContractAddress) || params.IsTrxHotWalletDeposit() {
			return s.bs.Wallets().Hot().ActivateWallet(ctx,
				params.checkResult.OwnerID,
				s.blockchain,
				*params.event.AddressTo,
				repos.WithTx(dbTx),
			)
		}
	}

	return nil
}

type eventCheck struct {
	address string
	kind    models.WebhookKind
}

func (s *scanner) checksForEvent(event *transactionsv2.Event) []eventCheck {
	checks := []eventCheck{}

	if event.AddressFrom != nil && *event.AddressFrom != "" {
		checks = append(checks, eventCheck{address: *event.AddressFrom, kind: models.WebhookKindTransfer})
	}

	if event.AddressTo != nil && *event.AddressTo != "" {
		checks = append(checks, eventCheck{address: *event.AddressTo, kind: models.WebhookKindDeposit})
	}

	if s.blockchain == wconstants.BlockchainTypeBitcoinCash {
		for _, check := range checks {
			addr, err := bch.DecodeAddressToCashAddr(check.address, s.sdk.BCH.ChainParams())
			if err == nil && addr != check.address {
				checks = append(checks, eventCheck{address: addr, kind: check.kind})
			}

			addr, err = bch.DecodeAddressToLegacyAddr(check.address, s.sdk.BCH.ChainParams())
			if err == nil && addr != check.address {
				checks = append(checks, eventCheck{address: addr, kind: check.kind})
			}
		}
	}

	return checks
}
