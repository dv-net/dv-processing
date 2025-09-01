package transfers

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/dv-net/dv-processing/rpccode"
	"golang.org/x/sync/errgroup"
)

// processLitecoin handle transfer request for btc like blockchains
func (s *Service) processBTCLike(ctx context.Context, req *CreateTransferRequest) error {
	switch req.Blockchain {
	case wconstants.BlockchainTypeBitcoin:
		if !s.config.Blockchain.Bitcoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeLitecoin:
		if !s.config.Blockchain.Litecoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeBitcoinCash:
		if !s.config.Blockchain.BitcoinCash.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	case wconstants.BlockchainTypeDogecoin:
		if !s.config.Blockchain.Dogecoin.Enabled {
			return rpccode.GetErrorByCode(rpccode.RPCCodeBlockchainIsDisabled)
		}
	default:
		return fmt.Errorf("unsupported blockchain: %s", req.Blockchain)
	}

	if !req.WholeAmount {
		return fmt.Errorf("only whole amount is supported for bitcoin transfers")
	}

	if !req.WholeAmount && len(req.FromAddresses) != 1 {
		return fmt.Errorf("only one from address is supported for transfer with amount")
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(10)

	// check balances
	for _, fromAddress := range req.FromAddresses {
		eg.Go(func() error {
			balance, err := s.eproxySvc.AddressBalance(egCtx, fromAddress, req.AssetIdentifier, req.Blockchain)
			if err != nil {
				return fmt.Errorf("get balance: %w", err)
			}

			// TODO: edit this condition
			if !req.WholeAmount && balance.LessThanOrEqual(req.Amount.Decimal) {
				return fmt.Errorf("%w for transfer. required: %s, available: %s", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance), req.Amount.Decimal, balance)
			}

			if req.WholeAmount && !balance.IsPositive() {
				return fmt.Errorf("%w for transfer with whole amount, available: 0", rpccode.GetErrorByCode(rpccode.RPCCodeAddressEmptyBalance))
			}

			// check active transfers with the same from address
			transfersInProcess, err := s.store.Transfers().Find(ctx, FindParams{
				StatusesIn:  []string{constants.TransferStatusProcessing.String()},
				FromAddress: &fromAddress,
				Blockchain:  &req.Blockchain,
				Limit:       utils.Pointer(1),
			})
			if err != nil {
				return fmt.Errorf("find transfers: %w", err)
			}

			if len(transfersInProcess) > 0 {
				return fmt.Errorf("%s: %w", fromAddress, rpccode.GetErrorByCode(rpccode.RPCCodeAddressIsTaken))
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("check balances on from addresses: %w", err)
	}

	return nil
}
