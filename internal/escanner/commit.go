package escanner

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/webhooks"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfer_transactions"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

// cursorUpdate describes how commitEvents advances the scanner cursor.
// A nil cursor commits events without touching processed_blocks — used by
// flows that replay history out of block order (address backfill).
type cursorUpdate struct {
	height int64
	hash   string
	exists bool // mirrors the existsLastBlockDB semantics: create the cursor row on the first commit
}

// txHook runs extra statements inside the commitEvents transaction.
type txHook func(ctx context.Context, dbTx pgx.Tx) error

// commitOptions controls what commitEvents writes besides the webhooks.
type commitOptions struct {
	// cursor advances processed_blocks; nil leaves the cursor untouched.
	cursor *cursorUpdate
	// hook runs inside the same transaction (e.g. backfill progress).
	hook txHook
	// finalized marks events from blocks that already have enough confirmations
	// (address backfill). Their webhooks skip the waiting_confirmations stage.
	finalized bool
}

// webhookPlan is what commitEvents writes for a single event.
type webhookPlan struct {
	status       models.WebhookEventStatus
	sendWebhook  bool // create a webhook now with status
	enqueueCheck bool // enqueue a confirmation-check job (it creates the completed webhook)
}

// planWebhook decides how an event is committed.
//
//   - live blocks: waiting_confirmations webhook + confirmation-check job;
//   - finalized blocks: completed webhook right away, no job;
//   - finalized EVM deposits: no webhook, only the job — it runs immediately and
//     keeps the deposit balance delta verification before the completed webhook.
//
// A finalized event that still lacks confirmations falls back to the live flow.
func planWebhook(blockchain wconstants.BlockchainType, kind models.WebhookKind, confirmations uint64, finalized bool) webhookPlan {
	live := webhookPlan{status: models.WebhookEventStatusWaitingConfirmations, sendWebhook: true, enqueueCheck: true}

	if !finalized || constants.ConfirmationsTimeout(blockchain, confirmations) > 0 {
		return live
	}

	if blockchain.IsEVM() && kind == models.WebhookKindDeposit {
		return webhookPlan{enqueueCheck: true}
	}

	return webhookPlan{status: models.WebhookEventStatusCompleted, sendWebhook: true}
}

// jobDedupeKey is WebhookWaitingConfirmationsArgs with the external wallet id
// compared by value instead of by pointer.
type jobDedupeKey struct {
	args             taskmanager.WebhookWaitingConfirmationsArgs
	externalWalletID string
}

func newJobDedupeKey(args taskmanager.WebhookWaitingConfirmationsArgs) jobDedupeKey {
	key := jobDedupeKey{args: args}
	if args.ExternalWalletID != nil {
		key.externalWalletID = *args.ExternalWalletID
	}
	key.args.ExternalWalletID = nil
	return key
}

// commitEvents writes webhook params to the database in a single transaction:
// webhooks, confirmation-check jobs (one bulk insert), the optional hook and the
// optional scanner cursor advance. All lookups run on the transaction connection,
// so a commit never holds one pool connection while waiting for another.
func (p *eventPipeline) commitEvents(ctx context.Context, whParams []createWebhookParams, opts commitOptions) error {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	now := time.Now()

	if err := pgx.BeginTxFunc(ctx, p.store.PSQLConn(), pgx.TxOptions{}, func(dbTx pgx.Tx) error {
		if len(whParams) != 0 {
			batchParams := make([]webhooks.BatchCreateParams, 0, len(whParams))
			jobs := make([]river.InsertManyParams, 0, len(whParams))
			seenJobs := make(map[jobDedupeKey]struct{}, len(whParams))

			for _, params := range whParams {
				plan := planWebhook(p.blockchain, params.whKind, params.tx.Confirmations, opts.finalized)

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
					Blockchain:       p.blockchain,
					Tx:               transactionData,
					Event:            transactionEventData,
					WebhookKind:      params.whKind,
					WalletType:       params.checkResult.WalletType,
					WebhookStatus:    plan.status,
					OwnerID:          params.checkResult.OwnerID,
					ExternalWalletID: params.checkResult.ExternalWalletID,
				}

				// post create webhook handler (system tx flag, tron activation)
				if err := p.beforeCreateWhHandler(ctx, dbTx, params, &whCreateParamsData); err != nil {
					return fmt.Errorf("post create webhook handler: %w", err)
				}

				if plan.sendWebhook {
					param, err := p.bs.Webhooks().EventTransactionCreateParams(ctx, whCreateParamsData, repos.WithTx(dbTx))
					if err != nil {
						return fmt.Errorf("get params for webhook: %w", err)
					}

					batchParams = append(batchParams, param)
				}

				if !plan.enqueueCheck {
					continue
				}

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

				args := taskmanager.WebhookWaitingConfirmationsArgs{
					Blockchain:             p.blockchain,
					Hash:                   params.tx.Hash,
					Address:                address,
					EventBlockchainUniqKey: blockchainUniqKey,
					WebhookKind:            params.whKind,
					WalletType:             params.checkResult.WalletType,
					OwnerID:                params.checkResult.OwnerID,
					IsSystem:               whCreateParamsData.IsSystem,
				}

				args.ExternalWalletID = params.checkResult.ExternalWalletID

				// the bulk insert resolves unique conflicts with ON CONFLICT DO UPDATE,
				// which fails when one statement hits the same key twice
				key := newJobDedupeKey(args)
				if _, ok := seenJobs[key]; ok {
					continue
				}
				seenJobs[key] = struct{}{}

				jobs = append(jobs, river.InsertManyParams{
					Args: args,
					InsertOpts: &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{
							ByArgs: true,
						},
						ScheduledAt: time.Now().Add(constants.ConfirmationsTimeout(p.blockchain, params.tx.Confirmations)),
					},
				})
			}

			if len(jobs) != 0 {
				if _, err := p.client.InsertManyTx(ctx, dbTx, jobs); err != nil {
					return fmt.Errorf("insert webhook jobs: %w", err)
				}
			}

			if len(batchParams) != 0 {
				if err := p.bs.Webhooks().BatchCreate(ctx, batchParams, repos.WithTx(dbTx)); err != nil {
					return err
				}
			}
		}

		if opts.hook != nil {
			if err := opts.hook(ctx, dbTx); err != nil {
				return fmt.Errorf("commit hook: %w", err)
			}
		}

		cursor := opts.cursor
		if cursor == nil {
			return nil
		}

		p.logger.Debugf("processed block %d in %s", cursor.height, time.Since(now))

		if cursor.exists {
			if err := p.bs.ProcessedBlocks().UpdateNumberWithHash(ctx, p.blockchain, cursor.height, cursor.hash, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("update block number: %w", err)
			}

			p.logger.Debugf("updated last block from explorer %s is %d", p.blockchain.String(), cursor.height)
		} else {
			if err := p.bs.ProcessedBlocks().Create(ctx, p.blockchain, cursor.height, cursor.hash, repos.WithTx(dbTx)); err != nil {
				return fmt.Errorf("create block number: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if opts.cursor != nil {
		p.parsedHeight.Store(opts.cursor.height)
	}

	return nil
}

// commitBlockResult writes a fetched block result to the database (must be called sequentially).
func (p *eventPipeline) commitBlockResult(ctx context.Context, r blockFetchResult, existsLastBlockDB bool) error {
	return p.commitEvents(ctx, r.whParams, commitOptions{
		cursor: &cursorUpdate{
			height: r.height,
			hash:   r.blockHash,
			exists: existsLastBlockDB,
		},
	})
}

// beforeCreateWhHandler - before create webhook handler
func (p *eventPipeline) beforeCreateWhHandler(
	ctx context.Context,
	dbTx pgx.Tx,
	params createWebhookParams,
	whCreateParamsData *webhooks.EventTransactionCreateParamsData,
) error {
	if !p.blockchain.IsSystemTransactionsSupported() {
		return nil
	}

	// Fetch system transactions with existing transfers by tx_hash
	systemTxs, err := p.store.TransferTransactions(repos.WithTx(dbTx)).FindSystemTransactions(ctx, repo_transfer_transactions.FindSystemTransactionsParams{
		OwnerID:       whCreateParamsData.OwnerID,
		Blockchain:    p.blockchain,
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

	if p.blockchain == wconstants.BlockchainTypeTron && !params.checkResult.Activated() {
		// Check if wallet activation is required
		if params.IsTrxContractActivationDeposit(p.conf.Tron.ActivationContractAddress) || params.IsTrxHotWalletDeposit() {
			return p.bs.Wallets().Hot().ActivateWallet(ctx,
				params.checkResult.OwnerID,
				p.blockchain,
				*params.event.AddressTo,
				repos.WithTx(dbTx),
			)
		}
	}

	return nil
}
