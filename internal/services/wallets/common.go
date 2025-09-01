package wallets

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/google/uuid"
)

func (s *Service) GetSequenceByWalletType(
	ctx context.Context,
	walletType constants.WalletType,
	ownerID uuid.UUID,
	blockchain wconstants.BlockchainType,
	address string,
) (int32, error) {
	if !walletType.Valid() {
		return 0, fmt.Errorf("invalid wallet type: %s", walletType.String())
	}

	if ownerID == uuid.Nil {
		return 0, fmt.Errorf("empty owner id")
	}

	if !blockchain.Valid() {
		return 0, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	if address == "" {
		return 0, fmt.Errorf("empty address")
	}

	var sequence int32
	switch walletType {
	case constants.WalletTypeHot:
		{
			hotWallet, err := s.Hot().Get(ctx, ownerID, blockchain, address)
			if err != nil {
				return 0, fmt.Errorf("get hot wallet: %w", err)
			}

			sequence = hotWallet.Sequence
		}
	case constants.WalletTypeProcessing:
		{
			processingWallet, err := s.Processing().GetByOwnerID(ctx, ownerID, blockchain, address)
			if err != nil {
				return 0, fmt.Errorf("get processing wallet: %w", err)
			}

			sequence = processingWallet.Sequence
		}
	default:
		return 0, fmt.Errorf("unsupported wallet type: %s", walletType)
	}

	return sequence, nil
}
