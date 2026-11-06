package eproxy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dv-net/dv-processing/internal/interceptors"

	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/pkg/retry"
	"github.com/dv-net/dv-processing/pkg/utils"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	addressesv2 "github.com/dv-net/dv-proto/gen/go/eproxy/addresses/v2"
	"github.com/dv-net/dv-proto/gen/go/eproxy/addresses/v2/addressesv2connect"
	assetsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/assets/v2"
	"github.com/dv-net/dv-proto/gen/go/eproxy/assets/v2/assetsv2connect"
	blockv2 "github.com/dv-net/dv-proto/gen/go/eproxy/blocks/v2"
	"github.com/dv-net/dv-proto/gen/go/eproxy/blocks/v2/blocksv2connect"
	"github.com/dv-net/dv-proto/gen/go/eproxy/btclike/v2/btclikev2connect"
	commonv2 "github.com/dv-net/dv-proto/gen/go/eproxy/common/v2"
	incidentsv2 "github.com/dv-net/dv-proto/gen/go/eproxy/incidents/v2"
	"github.com/dv-net/dv-proto/gen/go/eproxy/incidents/v2/incidentsv2connect"
	trxv2 "github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2"
	"github.com/dv-net/dv-proto/gen/go/eproxy/transactions/v2/transactionsv2connect"
	"github.com/dv-net/dv-proto/gen/go/eproxy/tron/v1/tronv1connect"
	"github.com/dv-net/dv-proto/go/eproxy"

	"connectrpc.com/connect"
	"github.com/dv-net/mx/clients/connectrpc_client"
	"github.com/shopspring/decimal"
)

type Service struct {
	eproxyClient *eproxy.Client
}

func New(ctx context.Context, cfg connectrpc_client.Config) (*Service, error) {
	processingID, ok := ctx.Value(constants.ProcessingIDParamName).(string)
	if !ok {
		return nil, fmt.Errorf("undefined processing ID")
	}
	processingVersion, ok := ctx.Value(constants.ProcessingVersionParamName).(string)
	if !ok {
		return nil, fmt.Errorf("undefined processing version")
	}

	connectRPCOptions := []connect.ClientOption{
		connect.WithInterceptors(eproxy.NewClientInterceptor(processingVersion)),
		connect.WithInterceptors(interceptors.NewProcessingIdentity(processingID, processingVersion)),
	}

	client, err := eproxy.NewClient(
		cfg.Addr,
		eproxy.WithVersion(processingVersion),
		eproxy.WithConnectrpcOpts(connectRPCOptions...),
	)
	if err != nil {
		return nil, fmt.Errorf("create eproxy client: %w", err)
	}

	svc := &Service{
		eproxyClient: client,
	}

	return svc, nil
}

// BTCLikeClient returns the bitcoin client
func (s *Service) BTCLikeClient() btclikev2connect.BtcLikeServiceClient {
	return s.eproxyClient.BTCLikeClient
}

// TronClient returns the tron client
func (s *Service) TronClient() tronv1connect.TronServiceClient { return s.eproxyClient.TronClient }

// AddressesClient returns the addresses client
func (s *Service) AddressesClient() addressesv2connect.AddressesServiceClient {
	return s.eproxyClient.AddressesClient
}

// BlocksClient returns the blocks client
func (s *Service) BlocksClient() blocksv2connect.BlocksServiceClient {
	return s.eproxyClient.BlocksClient
}

// AssetsClient returns the assets client
func (s *Service) AssetsClient() assetsv2connect.AssetsServiceClient {
	return s.eproxyClient.AssetsClient
}

// TransactionsClient returns the transactions client
func (s *Service) TransactionsClient() transactionsv2connect.TransactionsServiceClient {
	return s.eproxyClient.TransactionsClient
}

func (s *Service) IncidentsClient() incidentsv2connect.IncidentsServiceClient {
	return s.eproxyClient.IncidentsClient
}

// AddressBalances returns the balances of the address on the blockchain
func (s *Service) AddressBalances(ctx context.Context, address string, blockchain wconstants.BlockchainType) ([]*models.Asset, error) {
	srvBlockchain := ConvertBlockchain(blockchain)

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[addressesv2.InfoResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.eproxyClient.AddressesClient.Info(
			ctx, connect.NewRequest(&addressesv2.InfoRequest{
				Address:    address,
				Blockchain: srvBlockchain,
			}),
		)
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return nil, err
	}

	assets := make([]*models.Asset, 0, len(response.Msg.GetItem().GetAssets()))
	for _, asset := range response.Msg.GetItem().GetAssets() {
		amount, err := decimal.NewFromString(asset.GetAmount())
		if err != nil {
			return nil, err
		}

		assets = append(
			assets,
			&models.Asset{
				ID:     asset.GetAssetIdentifier(),
				Amount: amount,
			},
		)
	}

	return assets, nil
}

// AddressBalance returns the balance of the address for the asset on the blockchain
func (s *Service) AddressBalance(ctx context.Context, address, assetIdentifier string, blockchain wconstants.BlockchainType) (decimal.Decimal, error) {
	if address == "" {
		return decimal.Zero, ErrAddressRequired
	}

	if assetIdentifier == "" {
		return decimal.Zero, ErrAssetIdentifierRequired
	}

	if !blockchain.Valid() {
		return decimal.Zero, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[addressesv2.BalanceResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.eproxyClient.AddressesClient.Balance(
			ctx, connect.NewRequest(&addressesv2.BalanceRequest{
				Address:         address,
				AssetIdentifier: assetIdentifier,
				Blockchain:      ConvertBlockchain(blockchain),
			}),
		)
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return decimal.Decimal{}, err
	}

	balance, err := decimal.NewFromString(response.Msg.GetAmount())
	if err != nil {
		return decimal.Decimal{}, err
	}

	return balance, nil
}

// AddressBalanceAt returns the balance of the address for the asset on the blockchain at a specific block number.
func (s *Service) AddressBalanceAt(ctx context.Context, address, assetIdentifier string, blockchain wconstants.BlockchainType, blockNumber uint64) (decimal.Decimal, error) {
	if address == "" {
		return decimal.Zero, ErrAddressRequired
	}

	if assetIdentifier == "" {
		return decimal.Zero, ErrAssetIdentifierRequired
	}

	if !blockchain.Valid() {
		return decimal.Zero, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[addressesv2.BalanceResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.eproxyClient.AddressesClient.Balance(
			ctx, connect.NewRequest(&addressesv2.BalanceRequest{
				Address:         address,
				AssetIdentifier: assetIdentifier,
				Blockchain:      ConvertBlockchain(blockchain),
				BlockNumber:     &blockNumber,
			}),
		)
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return decimal.Decimal{}, err
	}

	balance, err := decimal.NewFromString(response.Msg.GetAmount())
	if err != nil {
		return decimal.Decimal{}, err
	}

	return balance, nil
}

// AssetDecimals returns the number of decimals for the asset on the blockchain
func (s *Service) AssetDecimals(ctx context.Context, blockchain wconstants.BlockchainType, assetIdentifier string) (int64, error) {
	if assetIdentifier == "" {
		return 0, ErrAssetIdentifierRequired
	}

	if !blockchain.Valid() {
		return 0, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[assetsv2.InfoResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.AssetsClient().Info(ctx, connect.NewRequest(&assetsv2.InfoRequest{
			Blockchain: ConvertBlockchain(blockchain),
			Identifier: assetIdentifier,
		}))
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return 0, fmt.Errorf("get asset info: %w", err)
	}

	if response.Msg.Decimals == nil {
		return 0, fmt.Errorf("decimals is nil for asset %s", assetIdentifier)
	}

	return int64(*response.Msg.Decimals), nil
}

// LastBlockNumber returns the last block number of the blockchain
func (s *Service) LastBlockNumber(ctx context.Context, blockchain wconstants.BlockchainType) (uint64, error) {
	srvBlockchain := ConvertBlockchain(blockchain)

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	response, err := s.eproxyClient.BlocksClient.LastBlockHeight(
		ctx,
		connect.NewRequest(&blockv2.LastBlockHeightRequest{
			Blockchain: srvBlockchain,
		}),
	)
	if err != nil {
		return 0, err
	}

	return response.Msg.GetHeight(), nil
}

// GetIncidents returns recent incidents for the blockchain
func (s *Service) GetIncidents(ctx context.Context, blockchain wconstants.BlockchainType, limit uint32) ([]*incidentsv2.Incident, error) {
	incidents, err := s.eproxyClient.IncidentsClient.Find(ctx, connect.NewRequest(&incidentsv2.FindRequest{
		Blockchain: ConvertBlockchain(blockchain),
		Common: &commonv2.FindRequestCommon{
			Page:     utils.Pointer(uint32(1)),
			PageSize: utils.Pointer(limit),
		},
	}))
	if err != nil {
		return nil, fmt.Errorf("get incidents: %w", err)
	}

	return incidents.Msg.GetItems(), nil
}

// FindRecentRollbackBlock returns the revert-to block height from the latest rollback
// incident not older than rollbackIncidentMaxAge. found is false when there is no such incident.
func (s *Service) FindRecentRollbackBlock(ctx context.Context, blockchain wconstants.BlockchainType) (block uint64, found bool, err error) {
	incidents, err := s.GetIncidents(ctx, blockchain, 10)
	if err != nil {
		return 0, false, err
	}

	maxAge := constants.RollbackIncidentMaxAge()

	for _, incident := range incidents {
		if incident.GetType() != incidentsv2.IncidentType_INCIDENT_TYPE_ROLLBACK {
			continue
		}

		if incident.GetCreatedAt() != nil && time.Since(incident.GetCreatedAt().AsTime()) > maxAge {
			break // incidents are ordered newest-first; all further are older
		}

		return incident.GetDataRollback().GetRevertToBlockHeight(), true, nil
	}

	return 0, false, nil
}

type FindTransactionsParams struct {
	BlockHeight     *uint64
	BlockHeightTo   *uint64
	FilterByAddress *bool
	Hash            *string
	Address         *string

	// PageRetryDelays retries a page that failed with a transient error
	// (deadline exceeded, unavailable, resource exhausted): one retry per delay.
	// Empty means no retries.
	PageRetryDelays []time.Duration
}

// IsTransientError reports eproxy errors worth retrying: the request may succeed
// when eproxy or the explorer behind it is less loaded.
func IsTransientError(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeDeadlineExceeded, connect.CodeUnavailable, connect.CodeResourceExhausted:
		return true
	}
	return false
}

// FindTransactions returns transactions by request on the blockchain
func (s *Service) FindTransactions(ctx context.Context, blockchain wconstants.BlockchainType, request FindTransactionsParams) ([]*trxv2.Transaction, error) {
	var (
		page     = uint32(1)
		pageSize = uint32(1000)
	)

	// eproxy caps page size at 50 for address queries (tron, evm explorers);
	// block queries are served with a larger page.
	if request.Address != nil {
		pageSize = 50
	}

	data := make([]*trxv2.Transaction, 0)

	var blockHeightOffset *uint64
	if request.BlockHeightTo != nil {
		if request.BlockHeight == nil || *request.BlockHeightTo < *request.BlockHeight {
			return nil, fmt.Errorf("invalid block height range: from=%v to=%v", request.BlockHeight, *request.BlockHeightTo)
		}
		blockHeightOffset = utils.Pointer(*request.BlockHeightTo - *request.BlockHeight)
	}

	req := &trxv2.FindRequest{
		BlockHeight:       request.BlockHeight,
		Hash:              request.Hash,
		Address:           request.Address,
		Blockchain:        ConvertBlockchain(blockchain),
		BlockHeightOffset: blockHeightOffset,
	}

	if request.FilterByAddress != nil {
		req.BitcoinLikeParams = &trxv2.FindRequest_BitcoinLikeParams{
			FilterByAddress: request.FilterByAddress,
		}
	}

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		req.Common = &commonv2.FindRequestCommon{Page: &page, PageSize: &pageSize}

		var response *connect.Response[trxv2.FindResponse]
		var err error
		for attempt := 0; ; attempt++ {
			reqCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
			response, err = s.eproxyClient.TransactionsClient.Find(reqCtx, connect.NewRequest(req))
			cancel()

			if err == nil || ctx.Err() != nil || attempt >= len(request.PageRetryDelays) || !IsTransientError(err) {
				break
			}

			select {
			case <-ctx.Done():
			case <-time.After(request.PageRetryDelays[attempt]):
			}
		}

		if err != nil {
			return nil, fmt.Errorf("error fetching page %d: %w", page, err)
		}

		items := response.Msg.GetItems()
		if len(items) == 0 {
			break
		}

		data = append(data, items...)

		if !response.Msg.NextPageExists {
			break
		}

		page++
	}

	return data, nil
}

// GetTransactionInfo returns transaction info by hash on the blockchain
func (s *Service) GetTransactionInfo(ctx context.Context, blockchain wconstants.BlockchainType, hash string) (*trxv2.Transaction, error) {
	if hash == "" {
		return nil, ErrHashIsRequired
	}

	if !blockchain.Valid() {
		return nil, fmt.Errorf("invalid blockchain type: %s", blockchain.String())
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var response *connect.Response[trxv2.GetInfoResponse]
	if err := retry.New().Do(func() error {
		var err error
		response, err = s.eproxyClient.TransactionsClient.GetInfo(
			ctx,
			connect.NewRequest(&trxv2.GetInfoRequest{
				Blockchain: ConvertBlockchain(blockchain),
				Hash:       hash,
			}),
		)
		if err != nil && !strings.Contains(err.Error(), errConnectionResetByPeer) {
			return fmt.Errorf("%w: %w", err, retry.ErrExit)
		}
		return err
	}); err != nil {
		return nil, err
	}

	if response.Msg == nil {
		return nil, fmt.Errorf("response message is nil for hash %s", hash)
	}

	tx := response.Msg.GetTransaction()

	if tx == nil {
		return nil, fmt.Errorf("transaction info is nil for hash %s", hash)
	}

	return tx, nil
}
