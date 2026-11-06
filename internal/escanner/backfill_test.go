package escanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
)

func TestAddressBackfillArgsValidate(t *testing.T) {
	valid := AddressBackfillArgs{
		IncidentID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		Blockchain: wconstants.BlockchainTypeBitcoin,
		Address:    "bc1qexample",
		FromBlock:  100,
		ToBlock:    200,
	}

	tests := []struct {
		name    string
		mutate  func(*AddressBackfillArgs)
		wantErr string
	}{
		{name: "valid"},
		{
			name:    "empty incident id",
			mutate:  func(a *AddressBackfillArgs) { a.IncidentID = "" },
			wantErr: "incident id is empty",
		},
		{
			name:    "invalid blockchain",
			mutate:  func(a *AddressBackfillArgs) { a.Blockchain = "unknown" },
			wantErr: "is invalid",
		},
		{
			name:    "empty address",
			mutate:  func(a *AddressBackfillArgs) { a.Address = "" },
			wantErr: "address is empty",
		},
		{
			name:    "negative from block",
			mutate:  func(a *AddressBackfillArgs) { a.FromBlock = -1 },
			wantErr: "is negative",
		},
		{
			name:    "to block before from block",
			mutate:  func(a *AddressBackfillArgs) { a.ToBlock = a.FromBlock - 1 },
			wantErr: "is less than from block",
		},
		{
			name:   "single block range",
			mutate: func(a *AddressBackfillArgs) { a.ToBlock = a.FromBlock },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := valid
			if tt.mutate != nil {
				tt.mutate(&args)
			}

			err := args.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestDedupeWebhookParams(t *testing.T) {
	key1 := "uniq-1"
	key2 := "uniq-2"

	tx1 := &transactionsv2.Transaction{Hash: "hash-1"}
	tx2 := &transactionsv2.Transaction{Hash: "hash-2"}

	params := []createWebhookParams{
		{tx: tx1, event: &transactionsv2.Event{BlockchainUniqKey: &key1}, whKind: models.WebhookKindDeposit},
		{tx: tx1, event: &transactionsv2.Event{BlockchainUniqKey: &key1}, whKind: models.WebhookKindDeposit},  // exact duplicate
		{tx: tx1, event: &transactionsv2.Event{BlockchainUniqKey: &key2}, whKind: models.WebhookKindDeposit},  // same tx, other event
		{tx: tx1, event: &transactionsv2.Event{BlockchainUniqKey: &key1}, whKind: models.WebhookKindTransfer}, // same event, other kind
		{tx: tx2, event: &transactionsv2.Event{BlockchainUniqKey: &key1}, whKind: models.WebhookKindDeposit},
	}

	got := dedupeWebhookParams(params)
	require.Len(t, got, 4)
	require.Equal(t, key2, *got[1].event.BlockchainUniqKey)
	require.Equal(t, models.WebhookKindTransfer, got[2].whKind)
	require.Equal(t, tx2, got[3].tx)

	require.Len(t, dedupeWebhookParams(nil), 0)
	single := dedupeWebhookParams(params[:1])
	require.Len(t, single, 1)
}

func TestParseBackfillProgress(t *testing.T) {
	const from, to = 100, 200

	tests := []struct {
		name     string
		metadata string
		want     backfillProgress
	}{
		{name: "no metadata", metadata: "", want: backfillProgress{BackfillNextBlock: from}},
		{name: "empty object", metadata: `{}`, want: backfillProgress{BackfillNextBlock: from}},
		{name: "other keys only", metadata: `{"unique_key_conflict":"x"}`, want: backfillProgress{BackfillNextBlock: from}},
		{name: "old format without size", metadata: `{"backfill_next_block":151}`, want: backfillProgress{BackfillNextBlock: 151}},
		{name: "saved progress", metadata: `{"backfill_next_block":151,"backfill_sub_range_blocks":7}`, want: backfillProgress{BackfillNextBlock: 151, BackfillSubRangeBlocks: 7}},
		{name: "range done", metadata: `{"backfill_next_block":201}`, want: backfillProgress{BackfillNextBlock: 201}},
		{name: "before range", metadata: `{"backfill_next_block":50,"backfill_sub_range_blocks":7}`, want: backfillProgress{BackfillNextBlock: from, BackfillSubRangeBlocks: 7}},
		{name: "past range", metadata: `{"backfill_next_block":500}`, want: backfillProgress{BackfillNextBlock: from}},
		{name: "wrong type", metadata: `{"backfill_next_block":"151"}`, want: backfillProgress{BackfillNextBlock: from}},
		{name: "broken json", metadata: `{`, want: backfillProgress{BackfillNextBlock: from}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseBackfillProgress([]byte(tt.metadata), from, to))
		})
	}
}

func TestSubRangeSizer(t *testing.T) {
	t.Run("starts at range/25", func(t *testing.T) {
		s := newSubRangeSizer(1, 25000, 0)
		require.Equal(t, uint64(1000), s.size)
		require.Equal(t, uint64(backfillMinSubRangeBlocks), s.minSize)
	})

	t.Run("resumes with saved size", func(t *testing.T) {
		require.Equal(t, uint64(250), newSubRangeSizer(1, 25000, 250).size)
		require.Equal(t, uint64(1000), newSubRangeSizer(1, 25000, 5000).size, "above max")
		require.Equal(t, uint64(1000), newSubRangeSizer(1, 25000, 10).size, "below min")
	})

	t.Run("short range", func(t *testing.T) {
		s := newSubRangeSizer(100, 110, 0)
		require.Equal(t, uint64(1), s.size)
		require.Equal(t, uint64(1), s.minSize)
		require.False(t, s.shrink())
	})

	t.Run("busy halves, quiet doubles, bounded", func(t *testing.T) {
		s := newSubRangeSizer(1, 25000, 0)

		s.observe(backfillBusyTxs)
		require.Equal(t, uint64(500), s.size)

		s.observe(backfillQuietTxs) // in between: unchanged
		require.Equal(t, uint64(500), s.size)

		s.observe(0)
		require.Equal(t, uint64(1000), s.size)
		s.observe(0)
		require.Equal(t, uint64(1000), s.size, "never above max")

		for range 10 {
			s.observe(backfillBusyTxs * 2)
		}
		require.Equal(t, uint64(backfillMinSubRangeBlocks), s.size, "never below min")
	})

	t.Run("shrink until min", func(t *testing.T) {
		s := newSubRangeSizer(1, 25000, 0)
		var shrinks int
		for s.shrink() {
			shrinks++
		}
		require.Equal(t, 4, shrinks) // 1000 -> 500 -> 250 -> 125 -> 100
		require.Equal(t, uint64(backfillMinSubRangeBlocks), s.size)
	})
}
