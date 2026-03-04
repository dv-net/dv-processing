package escanner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
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
	blocksv2 "github.com/dv-net/dv-proto/gen/go/eproxy/blocks/v2"
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
				lastDBBlockHeight, err := s.bs.EProxy().LastBlockNumber(ctx, s.blockchain)
				if err != nil {
					return fmt.Errorf("get last block from explorer: %w", err)
				}

				if s.lastNodeBlockHeight.Load() == int64(lastDBBlockHeight) { //nolint:gosec
					return nil
				}

				s.lastNodeBlockHeight.Store(int64(lastDBBlockHeight)) //nolint:gosec

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

// handleBlocks processes all new blocks using parallel fetching per chunk and sequential commits.
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

	blocksInChunk := int64(s.conf.EScanner.BlocksInChunk)
	if blocksInChunk <= 0 {
		blocksInChunk = 1
	}

	// Fetch last known hash once; updated in memory after each committed batch.
	var lastKnownHash string
	if existsLastBlockDB {
		lastBlock, err := s.bs.ProcessedBlocks().LastBlock(ctx, s.blockchain)
		if err == nil {
			lastKnownHash = lastBlock.Hash
		}
	}

	lag := lnBlock - lpBlock
	var catchUpStart time.Time
	var catchUpBlocksDone int64
	if lag > blocksInChunk {
		catchUpStart = time.Now()
		s.logger.Infof("catch-up started: lag=%d blocks (from=%d to=%d)", lag, lpBlock+1, lnBlock)
	}

	for batchStart := lpBlock + 1; batchStart <= lnBlock; batchStart += blocksInChunk {
		if ctx.Err() != nil {
			return nil //nolint:nilerr
		}

		batchEnd := min(batchStart+blocksInChunk-1, lnBlock)

		results, err := s.loadBlocks(ctx, batchStart, batchEnd)
		if err != nil {
			return fmt.Errorf("load blocks [%d, %d]: %w", batchStart, batchEnd, err)
		}

		// Validate chain integrity before committing — catches rollbacks within the batch too.
		if existsLastBlockDB {
			if err := s.validateChain(results, lastKnownHash); err != nil {
				s.logger.Warnf("chain validation failed: %s", err)
				return s.handleRollback(ctx)
			}
		}

		for _, r := range results {
			if err := s.commitBlockResult(ctx, r, existsLastBlockDB); err != nil {
				return fmt.Errorf("commit block %d: %w", r.height, err)
			}
			existsLastBlockDB = true
		}

		if len(results) > 0 {
			lastKnownHash = results[len(results)-1].blockHash
		}

		if !catchUpStart.IsZero() {
			catchUpBlocksDone += int64(len(results))
			elapsed := time.Since(catchUpStart)
			blocksPerSec := float64(catchUpBlocksDone) / elapsed.Seconds()
			remaining := lag - catchUpBlocksDone
			eta := time.Duration(float64(remaining)/blocksPerSec) * time.Second
			s.logger.Infof("catch-up progress: %d/%d blocks done (%.0f blocks/s, ETA %s)",
				catchUpBlocksDone, lag, blocksPerSec, eta.Round(time.Second))
		}
	}

	if !catchUpStart.IsZero() {
		elapsed := time.Since(catchUpStart)
		blocksPerSec := float64(lag) / elapsed.Seconds()
		s.logger.Infof("catch-up complete: %d blocks in %s (%.0f blocks/s)",
			lag, elapsed.Round(time.Millisecond), blocksPerSec)
	}

	time.Sleep(100 * time.Millisecond)

	return nil
}

// blockFetchResult holds data fetched for a single block (RPC only, no DB).
type blockFetchResult struct {
	height    int64
	blockHash string
	prevHash  string
	whParams  []createWebhookParams
}

// loadBlocks fetches a range of blocks in parallel (like evm LoadBlocks), preserving order.
func (s *scanner) loadBlocks(ctx context.Context, from, to int64) ([]blockFetchResult, error) {
	size := int(to - from + 1)
	results := make([]blockFetchResult, size)
	errs := make([]error, size)

	var wg sync.WaitGroup
	for i := range size {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = s.fetchBlockData(ctx, from+int64(idx))
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// validateChain checks that prevHash of each block matches the previous block's hash.
// Catches rollbacks both at the batch boundary and within the batch.
func (s *scanner) validateChain(results []blockFetchResult, lastKnownHash string) error {
	for i, r := range results {
		expected := lastKnownHash
		if i > 0 {
			expected = results[i-1].blockHash
		}
		if r.prevHash != "" && expected != "" && r.prevHash != expected {
			return fmt.Errorf("block %d: prevHash=%s expected=%s", r.height, r.prevHash, expected)
		}
	}
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

// fetchBlockData fetches all data for a single block via RPC (no DB operations).
func (s *scanner) fetchBlockData(ctx context.Context, blockHeight int64) (blockFetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	s.logger.Debugf("start fetching block %d", blockHeight)

	now := time.Now()
	txs, err := s.bs.EProxy().FindTransactions(ctx, s.blockchain, eproxy.FindTransactionsParams{
		BlockHeight: utils.Pointer(uint64(blockHeight)), //nolint:gosec
	})
	if err != nil {
		return blockFetchResult{}, fmt.Errorf("find transactions: %w", err)
	}

	s.logger.Debugf("found %d transactions in block %d in %s", len(txs), blockHeight, time.Since(now))

	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(blockHeight), //nolint:gosec
	}))
	if err != nil {
		return blockFetchResult{}, fmt.Errorf("get block for hash: %w", err)
	}

	blockHash := block.Msg.GetItem().GetHash()
	prevHash := block.Msg.GetItem().GetPrevHash()

	createWhParams := utils.NewSlice[createWebhookParams]()

	eg, gCtx := errgroup.WithContext(ctx)
	eg.SetLimit(100)

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
		return blockFetchResult{}, fmt.Errorf("wait for all transactions: %w", err)
	}

	return blockFetchResult{
		height:    blockHeight,
		blockHash: blockHash,
		prevHash:  prevHash,
		whParams:  createWhParams.GetAll(),
	}, nil
}

// commitBlockResult writes a fetched block result to the database (must be called sequentially).
func (s *scanner) commitBlockResult(ctx context.Context, r blockFetchResult, existsLastBlockDB bool) error {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	now := time.Now()

	if err := pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if len(r.whParams) != 0 {
			batchParams := make([]webhooks.BatchCreateParams, 0, len(r.whParams))
			for _, params := range r.whParams {
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

		s.logger.Debugf("processed block %d in %s", r.height, time.Since(now))

		if existsLastBlockDB {
			if err := s.bs.ProcessedBlocks().UpdateNumberWithHash(ctx, s.blockchain, r.height, r.blockHash, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("update block number: %w", err)
			}

			s.logger.Debugf("updated last block from explorer %s is %d", s.blockchain.String(), r.height)
		} else {
			if err := s.bs.ProcessedBlocks().Create(ctx, s.blockchain, r.height, r.blockHash, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("create block number: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	s.lastParsedBlockHeight.Store(r.height)

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

// handleRollback handles blockchain rollback by getting the new starting point from incidents API
func (s *scanner) handleRollback(ctx context.Context) error {
	s.logger.Infof("Handling rollback incident for blockchain %s", s.blockchain.String())

	newStartingBlock, err := s.bs.EProxy().GetRollbackStartingBlock(ctx, s.blockchain)
	if err != nil {
		return fmt.Errorf("get rollback starting block: %w", err)
	}

	s.logger.Infof("Rolling back to block %d for blockchain %s", newStartingBlock, s.blockchain.String())

	// The rollback target is the safe block we need to revert to (newStartingBlock - 1)
	// This is the last trusted block before the rollback occurred
	rollbackBlockHeight := int64(newStartingBlock) - 1 //nolint:gosec

	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(rollbackBlockHeight), //nolint:gosec
	}))
	if err != nil {
		return fmt.Errorf("get rollback block hash: %w", err)
	}

	rollbackBlockHash := block.Msg.GetItem().GetHash()

	// IMPORTANT: Set lastParsedBlockHeight to rollbackBlockHeight so the next iteration
	// will start parsing from newStartingBlock (rollbackBlockHeight + 1)
	// This ensures we re-parse blocks starting from newStartingBlock which were rolled back
	s.lastParsedBlockHeight.Store(rollbackBlockHeight)

	// Update the database with the rollback target block info
	return pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err := s.bs.ProcessedBlocks().UpdateNumberWithHash(ctx, s.blockchain, rollbackBlockHeight, rollbackBlockHash, repos.WithTx(dbTx)); err != nil {
			return fmt.Errorf("update processed block after rollback: %w", err)
		}

		s.logger.Infof("Successfully handled rollback, stored block %d with hash %s, will restart parsing from block %d", rollbackBlockHeight, rollbackBlockHash, newStartingBlock)
		return nil
	})
}
