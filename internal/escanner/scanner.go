package escanner

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	incidentsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/incidents/v2"
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

	// Check for potential rollback before processing next block
	if existsLastBlockDB && lpBlock >= 0 {
		nextBlock := lpBlock + 1
		s.logger.Debugf("Checking for rollback for next block %d", nextBlock)
		if err := s.checkForRollback(ctx, nextBlock); err != nil {
			return fmt.Errorf("rollback check for block %d: %w", nextBlock, err)
		}

		// Reload lpBlock in case it was updated during rollback recovery
		newLpBlock := s.lastParsedBlockHeight.Load()
		if newLpBlock != lpBlock {
			s.logger.Infof("Block height updated after rollback check: %d -> %d", lpBlock, newLpBlock)
			lpBlock = newLpBlock
		}
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
func (s *scanner) handleBlock(blockHeight int64, existsLastBlockDB bool) error { //nolint:funlen
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

	// Get block hash for rollback detection
	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(blockHeight), //nolint:gosec
	}))
	if err != nil {
		return fmt.Errorf("get block for hash: %w", err)
	}

	blockHash := block.Msg.GetItem().GetHash()

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
			if err := s.bs.ProcessedBlocks().UpdateNumberWithHash(ctx, s.blockchain, blockHeight, blockHash, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("update block number: %w", err)
			}

			s.logger.Debugf("updated last block from explorer %s is %d", s.blockchain.String(), blockHeight)
		} else {
			if err := s.bs.ProcessedBlocks().Create(ctx, s.blockchain, blockHeight, blockHash, repos.WithTx(dbTx)); err != nil {
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

// checkForRollback checks if the next block's prevHash matches our stored hash
func (s *scanner) checkForRollback(ctx context.Context, nextBlockHeight int64) error {
	s.logger.Debugf("Checking rollback for block %d", nextBlockHeight)

	// Get the next block to check prevHash consistency
	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(nextBlockHeight), //nolint:gosec
	}))
	if err != nil {
		// If we can't get the block, it might not exist yet OR there was a rollback
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			s.logger.Warnf("Block %d not found in explorer - checking for rollback incident", nextBlockHeight)
			// Check incidents as fallback - the block might have been rolled back
			return s.checkForIncidents(ctx)
		}
		return fmt.Errorf("get block for rollback check: %w", err)
	}

	// Get the previous block hash from the next block
	nextBlockPrevHash := block.Msg.GetItem().GetPrevHash()
	if nextBlockPrevHash == "" {
		s.logger.Debugf("block %d has no prevHash, skipping rollback check", nextBlockHeight)
		return nil
	}

	// Get stored block info from database
	lastBlock, err := s.bs.ProcessedBlocks().LastBlock(ctx, s.blockchain)
	if err != nil {
		if errors.Is(err, storecmn.ErrNotFound) {
			s.logger.Debugf("No stored block found, skipping rollback check")
			return nil
		}
		return fmt.Errorf("get last processed block: %w", err)
	}

	s.logger.Debugf("Comparing hashes: stored=%s, next_block_prevHash=%s", lastBlock.Hash, nextBlockPrevHash)

	// If we have a stored hash and the next block's prevHash doesn't match it, we have a rollback
	if lastBlock.Hash != "" && lastBlock.Hash != nextBlockPrevHash {
		s.logger.Warnf("Rollback detected! Stored hash %s != next block's prevHash %s for block %d",
			lastBlock.Hash, nextBlockPrevHash, nextBlockHeight)

		return s.handleRollback(ctx)
	}

	s.logger.Debugf("No rollback detected for block %d", nextBlockHeight)
	return nil
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

// checkForIncidents checks for new rollback incidents and processes them
func (s *scanner) checkForIncidents(ctx context.Context) error {
	s.logger.Debugf("Checking for new rollback incidents")

	// Get incidents from explorer
	incidents, err := s.bs.EProxy().GetIncidents(ctx, s.blockchain, 10)
	if err != nil {
		s.logger.Debugf("Failed to get incidents: %v", err)
		return nil // Don't fail scanner if we can't get incidents
	}

	if len(incidents) == 0 {
		s.logger.Debugf("No incidents found")
		return nil
	}

	// Check each incident
	for _, incident := range incidents {
		if incident.GetType() != incidentsv2.IncidentType_INCIDENT_TYPE_ROLLBACK {
			continue
		}

		// Check if this incident was already processed
		processed, err := s.bs.ProcessedIncidents().IsProcessed(ctx, s.blockchain, incident.GetId())
		if err != nil {
			return fmt.Errorf("check if incident processed: %w", err)
		}

		if processed {
			s.logger.Debugf("Incident %s already processed, skipping", incident.GetId())
			continue
		}

		// New incident found!
		rollbackStartBlock := int64(incident.GetDataRollback().GetRevertToBlockHeight()) //nolint:gosec
		currentBlock := s.lastParsedBlockHeight.Load()

		// Check if we need to rollback
		if currentBlock >= rollbackStartBlock { //nolint:nestif
			s.logger.Warnf("New rollback incident detected: id=%s, current_block=%d, rollback_to=%d",
				incident.GetId(), currentBlock, rollbackStartBlock-1)

			// Mark incident as processing before handling
			if err := s.bs.ProcessedIncidents().MarkAsProcessing(ctx,
				s.blockchain,
				incident.GetId(),
				"rollback",
				rollbackStartBlock,
				currentBlock); err != nil {
				s.logger.Errorf("Failed to mark incident as processing: %v", err)
			}

			// Handle the rollback
			if err := s.handleRollbackWithIncident(ctx, incident); err != nil {
				// Mark as failed
				if markErr := s.bs.ProcessedIncidents().MarkAsFailed(ctx,
					s.blockchain,
					incident.GetId(),
					err.Error()); markErr != nil {
					s.logger.Errorf("Failed to mark incident as failed: %v", markErr)
				}
				return fmt.Errorf("handle rollback for incident %s: %w", incident.GetId(), err)
			}

			// Mark as completed
			if err := s.bs.ProcessedIncidents().MarkAsCompleted(ctx, s.blockchain, incident.GetId()); err != nil {
				s.logger.Errorf("Failed to mark incident as completed: %v", err)
			}

			s.logger.Infof("Successfully processed rollback incident %s", incident.GetId())
			return nil // Process one incident at a time
		}

		s.logger.Debugf("Rollback incident %s not applicable: current_block=%d < rollback_start=%d",
			incident.GetId(), currentBlock, rollbackStartBlock)
	}

	return nil
}

// handleRollbackWithIncident handles rollback using incident information
func (s *scanner) handleRollbackWithIncident(ctx context.Context, incident *incidentsv2.Incident) error {
	rollbackStartBlock := int64(incident.GetDataRollback().GetRevertToBlockHeight()) //nolint:gosec
	rollbackBlockHeight := rollbackStartBlock - 1

	s.logger.Infof("Processing rollback incident %s: rolling back to block %d", incident.GetId(), rollbackBlockHeight)

	// Get the hash of the rollback target block
	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(rollbackBlockHeight), //nolint:gosec
	}))
	if err != nil {
		return fmt.Errorf("get rollback block hash: %w", err)
	}

	rollbackBlockHash := block.Msg.GetItem().GetHash()

	// Update lastParsedBlockHeight
	s.lastParsedBlockHeight.Store(rollbackBlockHeight)

	// Update database
	return pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err := s.bs.ProcessedBlocks().UpdateNumberWithHash(ctx,
			s.blockchain,
			rollbackBlockHeight,
			rollbackBlockHash,
			repos.WithTx(dbTx)); err != nil {
			return fmt.Errorf("update processed block: %w", err)
		}

		s.logger.Infof("Rollback completed: stored block %d with hash %s, will resume from block %d",
			rollbackBlockHeight, rollbackBlockHash, rollbackStartBlock)
		return nil
	})
}
