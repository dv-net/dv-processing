package escanner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_cold"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	blocksv2 "github.com/dv-net/dv-proto/gen/go/eproxy/blocks/v2"
	"github.com/dv-net/mx/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type scanner struct {
	*eventPipeline

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
	s := &scanner{}

	s.eventPipeline = newEventPipeline(
		logger.With(l,
			"module", "scanner",
			"blockchain", blockchain.String(),
		),
		c,
		blockchain,
		st,
		bs,
		sdk,
		&s.lastParsedBlockHeight,
	)
	s.SetRiverClient(tm.Client())

	return s
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

	// close out backfill incidents left over from previous runs; a failure here
	// must not prevent the scanner from starting
	if err := s.reconcileIncidents(ctx); err != nil {
		s.logger.Errorf("reconcile incidents: %s", err)
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

	if threshold := s.conf.BackfillThreshold(s.blockchain); threshold > 0 && lag > threshold {
		if h := constants.RollbackFallbackBlock(s.blockchain, lnBlock); h > lpBlock {
			s.logger.Warnf("lag %d exceeds backfill threshold %d: jumping to block %d", lag, threshold, h)
			if err := s.jump(ctx, lpBlock, h); err != nil {
				return fmt.Errorf("jump over %d blocks: %w", lag, err)
			}
			return nil // re-enter handleBlocks with the new cursor
		}
	}

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

	whParams, err := s.processTxEvents(ctx, txs)
	if err != nil {
		return blockFetchResult{}, err
	}

	return blockFetchResult{
		height:    blockHeight,
		blockHash: blockHash,
		prevHash:  prevHash,
		whParams:  whParams,
	}, nil
}

// rollbackStartingBlock returns the block to restart parsing from after a rollback.
// It uses a recent rollback incident from eproxy when the blockchain supports incidents,
// otherwise it falls back to lastParsedBlock - safeDepth.
func (s *scanner) rollbackStartingBlock(ctx context.Context) uint64 {
	if constants.SupportsRollbackIncidents(s.blockchain) {
		block, found, err := s.bs.EProxy().FindRecentRollbackBlock(ctx, s.blockchain)
		switch {
		case err != nil:
			s.logger.Warnf("get rollback incidents for blockchain %s, using fallback depth: %s", s.blockchain.String(), err)
		case found:
			return block
		default:
			s.logger.Infof("no recent rollback incident for blockchain %s, using fallback depth", s.blockchain.String())
		}
	}

	return uint64(constants.RollbackFallbackBlock(s.blockchain, s.lastParsedBlockHeight.Load())) //nolint:gosec
}

// handleRollback reverts the scanner to a safe block (see rollbackStartingBlock) and re-parses from there
func (s *scanner) handleRollback(ctx context.Context) error {
	s.logger.Infof("Handling rollback incident for blockchain %s", s.blockchain.String())

	newStartingBlock := s.rollbackStartingBlock(ctx)

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

// IncidentTypeAddressBackfill marks processed_incidents rows created by a scanner jump.
const IncidentTypeAddressBackfill = "address_backfill"

// jump moves the cursor to the finalized block h and enqueues an address backfill
// covering the skipped range (fromBlock, h]. The cursor update, the incident and the
// backfill jobs are committed in a single transaction, so a jump can never be lost.
func (s *scanner) jump(ctx context.Context, fromBlock, h int64) error {
	if h <= fromBlock {
		s.logger.Warnf("jump target %d is not ahead of the cursor %d, skipping", h, fromBlock)
		return nil
	}

	block, err := s.bs.EProxy().BlocksClient().Get(ctx, connect.NewRequest(&blocksv2.GetRequest{
		Blockchain: eproxy.ConvertBlockchain(s.blockchain),
		Height:     uint64(h), //nolint:gosec
	}))
	if err != nil {
		return fmt.Errorf("get block %d hash: %w", h, err)
	}
	hash := block.Msg.GetItem().GetHash()

	addresses, err := s.listWatchedAddresses(ctx)
	if err != nil {
		return fmt.Errorf("list watched addresses: %w", err)
	}

	incidentID := uuid.NewString()

	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if err := s.bs.ProcessedBlocks().UpdateNumberWithHash(ctx, s.blockchain, h, hash, repos.WithTx(dbTx)); err != nil {
			return fmt.Errorf("update processed block: %w", err)
		}

		if err := s.bs.ProcessedIncidents().MarkAsProcessing(ctx, s.blockchain, incidentID, IncidentTypeAddressBackfill, fromBlock+1, h, repos.WithTx(dbTx)); err != nil {
			return fmt.Errorf("mark incident as processing: %w", err)
		}

		const chunkSize = 500
		for start := 0; start < len(addresses); start += chunkSize {
			end := min(start+chunkSize, len(addresses))

			params := make([]river.InsertManyParams, 0, end-start)
			for _, address := range addresses[start:end] {
				params = append(params, river.InsertManyParams{
					Args: AddressBackfillArgs{
						IncidentID: incidentID,
						Blockchain: s.blockchain,
						Address:    address,
						FromBlock:  fromBlock + 1,
						ToBlock:    h,
					},
					InsertOpts: &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{
							ByArgs: true,
						},
					},
				})
			}

			if _, err := s.client.InsertManyTx(ctx, dbTx, params); err != nil {
				return fmt.Errorf("insert backfill jobs: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.lastParsedBlockHeight.Store(h)

	s.logger.Infof("jumped from block %d to %d (incident %s): %d backfill jobs enqueued", fromBlock, h, incidentID, len(addresses))

	return nil
}

// listWatchedAddresses returns every address known to CheckWallet for the blockchain.
// The listing is keyset-paginated: the wallets tables can hold tens of thousands
// of addresses per blockchain.
func (s *scanner) listWatchedAddresses(ctx context.Context) ([]string, error) {
	const pageSize = 5000

	seen := make(map[string]struct{})
	out := make([]string, 0)

	load := func(fetch func(ctx context.Context, after string, limit int32) ([]string, error)) error {
		after := ""
		for {
			addresses, err := fetch(ctx, after, pageSize)
			if err != nil {
				return err
			}

			for _, address := range addresses {
				if _, ok := seen[address]; ok {
					continue
				}

				seen[address] = struct{}{}
				out = append(out, address)
			}

			if len(addresses) < pageSize {
				return nil
			}

			after = addresses[len(addresses)-1]
		}
	}

	if err := load(func(ctx context.Context, after string, limit int32) ([]string, error) {
		return s.store.Wallets().Hot().GetAddressesByBlockchain(ctx, s.blockchain, after, limit)
	}); err != nil {
		return nil, fmt.Errorf("hot wallets: %w", err)
	}

	if err := load(func(ctx context.Context, after string, limit int32) ([]string, error) {
		return s.store.Wallets().Cold().GetAddressesByBlockchain(ctx, repo_wallets_cold.GetAddressesByBlockchainParams{
			Blockchain: s.blockchain,
			Address:    after,
			Limit:      limit,
		})
	}); err != nil {
		return nil, fmt.Errorf("cold wallets: %w", err)
	}

	if err := load(func(ctx context.Context, after string, limit int32) ([]string, error) {
		return s.store.Wallets().Processing().GetAddressesByBlockchain(ctx, s.blockchain, after, limit)
	}); err != nil {
		return nil, fmt.Errorf("processing wallets: %w", err)
	}

	return out, nil
}
