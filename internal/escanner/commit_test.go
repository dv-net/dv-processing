package escanner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/taskmanager"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func TestPlanWebhook(t *testing.T) {
	live := webhookPlan{status: models.WebhookEventStatusWaitingConfirmations, sendWebhook: true, enqueueCheck: true}
	completed := webhookPlan{status: models.WebhookEventStatusCompleted, sendWebhook: true}
	checkOnly := webhookPlan{enqueueCheck: true}

	tests := []struct {
		name          string
		blockchain    wconstants.BlockchainType
		kind          models.WebhookKind
		confirmations uint64
		finalized     bool
		want          webhookPlan
	}{
		{"scanner tron deposit", wconstants.BlockchainTypeTron, models.WebhookKindDeposit, 1, false, live},
		{"scanner ignores confirmations", wconstants.BlockchainTypeTron, models.WebhookKindDeposit, 100000, false, live},
		{"backfill tron deposit", wconstants.BlockchainTypeTron, models.WebhookKindDeposit, 100000, true, completed},
		{"backfill btc transfer", wconstants.BlockchainTypeBitcoin, models.WebhookKindTransfer, 1000, true, completed},
		{"backfill evm transfer", wconstants.BlockchainTypeEthereum, models.WebhookKindTransfer, 1000, true, completed},
		{"backfill evm deposit keeps balance check", wconstants.BlockchainTypeEthereum, models.WebhookKindDeposit, 1000, true, checkOnly},
		{"backfill without enough confirmations", wconstants.BlockchainTypeTron, models.WebhookKindDeposit, 1, true, live},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, planWebhook(tt.blockchain, tt.kind, tt.confirmations, tt.finalized))
		})
	}
}

func TestJobDedupeKey(t *testing.T) {
	ext1, ext2, ext3 := "wallet-1", "wallet-1", "wallet-2"
	base := taskmanager.WebhookWaitingConfirmationsArgs{
		Blockchain:  wconstants.BlockchainTypeTron,
		Hash:        "hash",
		Address:     "addr",
		WebhookKind: models.WebhookKindDeposit,
		OwnerID:     uuid.New(),
	}

	a, b, c := base, base, base
	a.ExternalWalletID = &ext1
	b.ExternalWalletID = &ext2 // same value, other pointer
	c.ExternalWalletID = &ext3

	require.Equal(t, newJobDedupeKey(a), newJobDedupeKey(b))
	require.NotEqual(t, newJobDedupeKey(a), newJobDedupeKey(c))
	require.NotEqual(t, newJobDedupeKey(a), newJobDedupeKey(base))
	require.NotNil(t, a.ExternalWalletID, "key must not mutate args")
}
