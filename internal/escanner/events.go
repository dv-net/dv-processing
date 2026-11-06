package escanner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/bch"
	"github.com/dv-net/dv-processing/pkg/walletsdk/tron"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	transactionsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
)

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

// processTxEvents filters the transfer events of the given transactions down to
// those that touch wallets known to us and prepares webhook params for them.
func (p *eventPipeline) processTxEvents(ctx context.Context, txs []*transactionsv2.Transaction) ([]createWebhookParams, error) {
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

					for _, check := range p.checksForEvent(event) {
						// check if the address belongs to us
						result, err := p.bs.Wallets().CheckWallet(gCtx, p.blockchain, check.address)
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
		return nil, fmt.Errorf("wait for all transactions: %w", err)
	}

	return createWhParams.GetAll(), nil
}

type eventCheck struct {
	address string
	kind    models.WebhookKind
}

func (p *eventPipeline) checksForEvent(event *transactionsv2.Event) []eventCheck {
	checks := []eventCheck{}

	if event.AddressFrom != nil && *event.AddressFrom != "" {
		checks = append(checks, eventCheck{address: *event.AddressFrom, kind: models.WebhookKindTransfer})
	}

	if event.AddressTo != nil && *event.AddressTo != "" {
		checks = append(checks, eventCheck{address: *event.AddressTo, kind: models.WebhookKindDeposit})
	}

	if p.blockchain == wconstants.BlockchainTypeBitcoinCash {
		for _, check := range checks {
			addr, err := bch.DecodeAddressToCashAddr(check.address, p.sdk.BCH.ChainParams())
			if err == nil && addr != check.address {
				checks = append(checks, eventCheck{address: addr, kind: check.kind})
			}

			addr, err = bch.DecodeAddressToLegacyAddr(check.address, p.sdk.BCH.ChainParams())
			if err == nil && addr != check.address {
				checks = append(checks, eventCheck{address: addr, kind: check.kind})
			}
		}
	}

	return checks
}
