package escanner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dv-net/mx/logger"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/dv-net/dv-processing/internal/config"
	"github.com/dv-net/dv-processing/internal/eproxy"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/store"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

const JobKindAddressBackfill = "address_backfill"

// AddressBackfillArgs replays the transactions of one watched address over a
// closed block range [FromBlock, ToBlock] that the scanner skipped (jump).
type AddressBackfillArgs struct {
	IncidentID string                    `json:"incident_id"`
	Blockchain wconstants.BlockchainType `json:"blockchain"`
	Address    string                    `json:"address"`
	FromBlock  int64                     `json:"from_block"`
	ToBlock    int64                     `json:"to_block"`
}

func (a AddressBackfillArgs) Kind() string { return JobKindAddressBackfill }

// QueueAddressBackfill is a dedicated river queue: backfill jobs are enqueued in
// bulk (one per watched address), and running all of them at once overloads
// eproxy and takes the shared workers from the confirmation-check jobs.
const (
	QueueAddressBackfill           = "address_backfill"
	addressBackfillQueueMaxWorkers = 5
)

// InsertOpts routes backfill jobs to their own queue.
func (a AddressBackfillArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueAddressBackfill}
}

func (a AddressBackfillArgs) Validate() error {
	if a.IncidentID == "" {
		return fmt.Errorf("incident id is empty")
	}

	if !a.Blockchain.Valid() {
		return fmt.Errorf("blockchain %s is invalid", a.Blockchain)
	}

	if a.Address == "" {
		return fmt.Errorf("address is empty")
	}

	if a.FromBlock < 0 {
		return fmt.Errorf("from block %d is negative", a.FromBlock)
	}

	if a.ToBlock < a.FromBlock {
		return fmt.Errorf("to block %d is less than from block %d", a.ToBlock, a.FromBlock)
	}

	return nil
}

// AddressBackfillWorker closes a skipped block range by pulling the watched
// address transactions from eproxy and pushing them through the shared event
// pipeline. It never touches the processed_blocks cursor.
type AddressBackfillWorker struct {
	logger logger.Logger
	river.WorkerDefaults[AddressBackfillArgs]

	conf  config.Blockchain
	store store.IStore
	bs    baseservices.IBaseServices
	sdk   *walletsdk.SDK

	mu        sync.RWMutex
	client    *river.Client[pgx.Tx]
	pipelines map[wconstants.BlockchainType]*eventPipeline
}

func NewAddressBackfillWorker(
	l logger.Logger,
	conf config.Blockchain,
	st store.IStore,
	bs baseservices.IBaseServices,
	sdk *walletsdk.SDK,
) *AddressBackfillWorker {
	w := &AddressBackfillWorker{
		logger:    l,
		conf:      conf,
		store:     st,
		bs:        bs,
		sdk:       sdk,
		pipelines: make(map[wconstants.BlockchainType]*eventPipeline, len(conf.Available())),
	}

	for _, chain := range conf.Available() {
		w.pipelines[chain] = newEventPipeline(
			logger.With(l, "module", "address-backfill", "blockchain", chain.String()),
			conf,
			chain,
			st,
			bs,
			sdk,
			new(atomic.Int64),
		)
	}

	return w
}

// Register implements taskmanager.WorkerWithRiverClient.
func (w *AddressBackfillWorker) Register(workers *river.Workers) error {
	return river.AddWorkerSafely(workers, w)
}

// Queues implements taskmanager.WorkerWithRiverClient.
func (w *AddressBackfillWorker) Queues() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		QueueAddressBackfill: {MaxWorkers: addressBackfillQueueMaxWorkers},
	}
}

// SetRiverClient implements taskmanager.WorkerWithRiverClient.
func (w *AddressBackfillWorker) SetRiverClient(client *river.Client[pgx.Tx]) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.client = client
	for _, p := range w.pipelines {
		p.SetRiverClient(client)
	}
}

func (w *AddressBackfillWorker) pipeline(blockchain wconstants.BlockchainType) *eventPipeline {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.pipelines[blockchain]
}

func (w *AddressBackfillWorker) Work(ctx context.Context, job *river.Job[AddressBackfillArgs]) error {
	if err := job.Args.Validate(); err != nil {
		return fmt.Errorf("validate args: %w", err)
	}

	p := w.pipeline(job.Args.Blockchain)
	if p == nil {
		return fmt.Errorf("no pipeline for blockchain %s", job.Args.Blockchain)
	}

	if p.client == nil {
		return fmt.Errorf("river client is not set")
	}

	if !w.isValidAddress(job.Args.Blockchain, job.Args.Address) {
		w.logger.Errorf("backfill cancelled: invalid %s address %q (incident=%s)", job.Args.Blockchain, job.Args.Address, job.Args.IncidentID)
		return river.JobCancel(fmt.Errorf("invalid %s address %q", job.Args.Blockchain, job.Args.Address))
	}

	from := uint64(job.Args.FromBlock) //nolint:gosec
	to := uint64(job.Args.ToBlock)     //nolint:gosec

	// a retry resumes from the first sub-range that was not committed yet,
	// with the sub-range size learned by the previous attempt
	progress := parseBackfillProgress(job.Metadata, from, to)
	sizer := newSubRangeSizer(from, to, progress.BackfillSubRangeBlocks)

	if progress.BackfillNextBlock > from {
		w.logger.Infof("backfill resumed: incident=%s address=%s range=[%d, %d] resume=%d sub-range=%d blocks",
			job.Args.IncidentID, job.Args.Address, from, to, progress.BackfillNextBlock, sizer.size)
	} else {
		w.logger.Infof("backfill started: incident=%s address=%s range=[%d, %d] sub-range=%d blocks",
			job.Args.IncidentID, job.Args.Address, from, to, sizer.size)
	}

	var (
		totalTxs  int
		subRanges int
		lastLog   = time.Now()
	)

	for start := progress.BackfillNextBlock; start <= to; {
		end := min(start+sizer.size-1, to)

		txs, err := w.bs.EProxy().FindTransactions(ctx, job.Args.Blockchain, eproxy.FindTransactionsParams{
			Address:         &job.Args.Address,
			BlockHeight:     &start,
			BlockHeightTo:   &end,
			FilterByAddress: utils.Pointer(true),
			PageRetryDelays: backfillPageRetryDelays,
		})
		if err != nil {
			// eproxy keeps timing out on this sub-range: retry it with a smaller
			// one instead of failing the job (deep pages are the slow ones)
			if eproxy.IsTransientError(err) && ctx.Err() == nil && sizer.shrink() {
				w.logger.Warnf("backfill: incident=%s address=%s blocks=[%d,%d]: %s; retrying with sub-range=%d blocks",
					job.Args.IncidentID, job.Args.Address, start, end, err, sizer.size)
				continue
			}
			return fmt.Errorf("find transactions [%d, %d]: %w", start, end, err)
		}
		totalTxs += len(txs)

		whParams, err := p.processTxEvents(ctx, txs)
		if err != nil {
			return fmt.Errorf("process transactions [%d, %d]: %w", start, end, err)
		}

		sizer.observe(len(txs))

		// webhooks and the progress mark are committed together: a retry never
		// re-sends webhooks of a sub-range that is already committed
		if err := p.commitEvents(ctx, dedupeWebhookParams(whParams), commitOptions{
			hook:      saveBackfillProgress(job.ID, end+1, sizer.size),
			finalized: true,
		}); err != nil {
			return fmt.Errorf("commit events [%d, %d]: %w", start, end, err)
		}

		subRanges++
		start = end + 1

		// throttled: a busy address would otherwise log every few seconds
		if time.Since(lastLog) >= backfillProgressLogInterval || start > to {
			lastLog = time.Now()
			w.logger.Infof("backfill progress: incident=%s address=%s blocks=%d/%d (%.1f%%) sub-ranges=%d sub-range=%d blocks txs=%d",
				job.Args.IncidentID, job.Args.Address, end-from+1, to-from+1, float64(end-from+1)*100/float64(to-from+1),
				subRanges, sizer.size, totalTxs)
		}
	}

	w.completeIncidentIfDone(ctx, job)

	w.logger.Infof("backfill complete: incident=%s address=%s txs=%d", job.Args.IncidentID, job.Args.Address, totalTxs)

	return nil
}

// isValidAddress guards against corrupted address rows: eproxy would otherwise
// reject them on every attempt and the job would retry until discard.
func (w *AddressBackfillWorker) isValidAddress(blockchain wconstants.BlockchainType, address string) bool {
	switch blockchain {
	case wconstants.BlockchainTypeTron:
		return w.sdk.Tron.ValidateAddress(address)
	case wconstants.BlockchainTypeBitcoin:
		return w.sdk.BTC.ValidateAddress(address)
	case wconstants.BlockchainTypeLitecoin:
		return w.sdk.LTC.ValidateAddress(address)
	case wconstants.BlockchainTypeBitcoinCash:
		return w.sdk.BCH.ValidateAddress(address)
	case wconstants.BlockchainTypeDogecoin:
		return w.sdk.Doge.ValidateAddress(address)
	case wconstants.BlockchainTypeEthereum,
		wconstants.BlockchainTypeBinanceSmartChain,
		wconstants.BlockchainTypePolygon,
		wconstants.BlockchainTypeArbitrum,
		wconstants.BlockchainTypeOptimism,
		wconstants.BlockchainTypeLinea:
		return w.sdk.EVM.ValidateAddress(address)
	}

	return true
}

// completeIncidentIfDone marks the incident as completed once its last active
// backfill job finishes. The caller's own job is still running at this point,
// so it is excluded from the count. If the last jobs finish concurrently, the
// incident is left in 'processing' and closed by the scanner's startup
// reconciliation instead.
func (w *AddressBackfillWorker) completeIncidentIfDone(ctx context.Context, job *river.Job[AddressBackfillArgs]) {
	counts, err := loadIncidentJobCounts(ctx, w.store.PSQLConn(), job.Args.IncidentID, job.ID)
	if err != nil {
		w.logger.Warnf("incident %s: count backfill jobs: %s", job.Args.IncidentID, err)
		return
	}

	done := counts["completed"] + counts["cancelled"]
	w.logger.Infof("incident %s: backfill progress %d/%d jobs done (%d active, %d discarded)",
		job.Args.IncidentID, done, counts.total(), counts.active(), counts["discarded"])

	if counts.active() > 0 {
		return
	}

	if err := w.bs.ProcessedIncidents().MarkAsCompleted(ctx, job.Args.Blockchain, job.Args.IncidentID); err != nil {
		w.logger.Warnf("incident %s: mark completed: %s", job.Args.IncidentID, err)
		return
	}

	w.logger.Infof("incident %s completed: all backfill jobs finished", job.Args.IncidentID)
}

// dedupeWebhookParams drops params that share the same transaction, event and
// webhook kind. The same transaction may be returned several times when it
// touches more than one watched address.
func dedupeWebhookParams(params []createWebhookParams) []createWebhookParams {
	if len(params) < 2 {
		return params
	}

	seen := make(map[string]struct{}, len(params))
	out := make([]createWebhookParams, 0, len(params))
	for _, p := range params {
		key := p.tx.GetHash() + "|" + p.event.GetBlockchainUniqKey() + "|" + string(p.whKind)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		out = append(out, p)
	}

	return out
}

// Sub-range sizing. eproxy paginates address queries page by page from the range
// start, so deep pages of a busy address get slow and can exceed the request
// deadline; every sub-range is also committed in one transaction. The size
// adapts to the address: it starts at range/backfillSubRanges, halves when a
// sub-range is busy or times out and doubles back when it is quiet.
const (
	backfillSubRanges         = 25
	backfillMinSubRangeBlocks = 100
	backfillBusyTxs           = 1000 // ~20 eproxy pages
	backfillQuietTxs          = 100  // ~2 eproxy pages

	backfillProgressLogInterval = 15 * time.Second
)

// backfillPageRetryDelays retries a page that failed with a transient eproxy error.
var backfillPageRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}

// subRangeSizer holds the current sub-range length in blocks.
type subRangeSizer struct {
	size, minSize, maxSize uint64
}

// newSubRangeSizer starts at saved (a size learned by a previous attempt) when it
// is within bounds, otherwise at the largest size.
func newSubRangeSizer(from, to, saved uint64) *subRangeSizer {
	maxSize := max((to-from+1)/backfillSubRanges, 1)
	s := &subRangeSizer{
		size:    maxSize,
		minSize: min(backfillMinSubRangeBlocks, maxSize),
		maxSize: maxSize,
	}
	if saved >= s.minSize && saved <= s.maxSize {
		s.size = saved
	}
	return s
}

// observe adapts the size to the number of transactions the last sub-range had.
func (s *subRangeSizer) observe(txs int) {
	switch {
	case txs >= backfillBusyTxs:
		s.size = max(s.size/2, s.minSize)
	case txs < backfillQuietTxs:
		s.size = min(s.size*2, s.maxSize)
	}
}

// shrink halves the size after a failed sub-range; false when it is already minimal.
func (s *subRangeSizer) shrink() bool {
	if s.size <= s.minSize {
		return false
	}
	s.size = max(s.size/2, s.minSize)
	return true
}

// backfillProgress is the resume point kept in the river job metadata. River
// merges metadata on state changes, so it survives job errors and retries.
type backfillProgress struct {
	BackfillNextBlock      uint64 `json:"backfill_next_block"`       // first block of the next sub-range
	BackfillSubRangeBlocks uint64 `json:"backfill_sub_range_blocks"` // sub-range size learned so far
}

const saveBackfillProgressQuery = `UPDATE river_job
SET metadata = metadata || jsonb_build_object('backfill_next_block', $2::bigint, 'backfill_sub_range_blocks', $3::bigint)
WHERE id = $1`

// saveBackfillProgress stores the resume point in the job metadata inside the
// transaction that commits the sub-range webhooks.
func saveBackfillProgress(jobID int64, nextBlock, subRangeBlocks uint64) txHook {
	return func(ctx context.Context, dbTx pgx.Tx) error {
		if _, err := dbTx.Exec(ctx, saveBackfillProgressQuery, jobID, int64(nextBlock), int64(subRangeBlocks)); err != nil { //nolint:gosec
			return fmt.Errorf("save backfill progress: %w", err)
		}
		return nil
	}
}

// parseBackfillProgress reads the saved progress. BackfillNextBlock falls back to from
// when nothing valid is saved; to+1 means the whole range is done.
func parseBackfillProgress(metadata []byte, from, to uint64) backfillProgress {
	var saved backfillProgress
	if len(metadata) == 0 || json.Unmarshal(metadata, &saved) != nil {
		return backfillProgress{BackfillNextBlock: from}
	}

	if saved.BackfillNextBlock < from || saved.BackfillNextBlock > to+1 {
		saved.BackfillNextBlock = from
	}

	return saved
}
