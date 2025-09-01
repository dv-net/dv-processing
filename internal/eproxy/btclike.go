package eproxy

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	btclikev2 "github.com/dv-net/dv-proto/gen/go/eproxy/btclike/v2"
)

// GetUTXO returns the UTXO data for the given address.
func (s *Service) GetUTXO(ctx context.Context, blockchain wconstants.BlockchainType, address string) ([]*btclikev2.UTXOResponse_Item, error) {
	if address == "" {
		return nil, ErrAddressRequired
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[btclikev2.UTXOResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.eproxyClient.BTCLikeClient.UTXO(ctx, connect.NewRequest(&btclikev2.UTXORequest{
			Address:    address,
			Blockchain: ConvertBlockchain(blockchain),
		}))
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return nil, err
	}

	if response.Msg == nil {
		return nil, fmt.Errorf("empty response")
	}

	return response.Msg.Items, nil
}
