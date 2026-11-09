package transfers

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
)

func transferEvent(from, to, asset, value string) *transactionsv2.Event {
	t := transactionsv2.EventType_EVENT_TYPE_TRANSFER
	ev := &transactionsv2.Event{Type: &t, AssetIdentifier: &asset}
	if from != "" {
		ev.AddressFrom = &from
	}
	if to != "" {
		ev.AddressTo = &to
	}
	if value != "" {
		ev.Value = &value
	}
	return ev
}

func TestDepositAndSenders(t *testing.T) {
	const deposit = "deposit-address"

	t.Run("evm: tx sender and transfer sender", func(t *testing.T) {
		from := "0xSender"
		tx := &transactionsv2.Transaction{
			AddressFrom: &from,
			Events: []*transactionsv2.Event{
				transferEvent("0xSender", "DEPOSIT-ADDRESS", "usdt", "10"),
				transferEvent("0xOther", "0xNotOurs", "usdt", "5"),
			},
		}

		deposited, senders := depositAndSenders(wconstants.BlockchainTypeEthereum, tx, deposit, "USDT")
		assert.True(t, deposited.Equal(decimal.NewFromInt(10)))
		assert.Equal(t, []string{"0xSender", "0xSender"}, senders)
	})

	t.Run("btc: every input address is a sender", func(t *testing.T) {
		tx := &transactionsv2.Transaction{
			Events: []*transactionsv2.Event{
				transferEvent("bc1qinput1", "", "btc", "0.3"),
				transferEvent("bc1qinput2", "", "btc", "0.2"),
				transferEvent("bc1qinput1", "", "btc", "0.1"),
				transferEvent("", deposit, "btc", "0.4"),
				transferEvent("", "bc1qinput1", "btc", "0.19"), // change
			},
		}

		deposited, senders := depositAndSenders(wconstants.BlockchainTypeBitcoin, tx, deposit, "btc")
		assert.True(t, deposited.Equal(decimal.RequireFromString("0.4")))
		assert.Equal(t, []string{"bc1qinput1", "bc1qinput2"}, senders)
	})

	t.Run("evm: an event without the receiver is not a sender", func(t *testing.T) {
		tx := &transactionsv2.Transaction{
			Events: []*transactionsv2.Event{
				transferEvent("0xSender", "", "eth", "1"),
				transferEvent("0xPayer", deposit, "eth", "1"),
			},
		}

		_, senders := depositAndSenders(wconstants.BlockchainTypeEthereum, tx, deposit, "eth")
		assert.Equal(t, []string{"0xPayer"}, senders)
	})
}
