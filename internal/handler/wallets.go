package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pquerna/otp/totp"

	"connectrpc.com/connect"
	commonv1 "github.com/dv-net/dv-processing/api/processing/common/v1"
	walletv1 "github.com/dv-net/dv-processing/api/processing/wallet/v1"
	"github.com/dv-net/dv-processing/api/processing/wallet/v1/walletv1connect"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/wallets"
	"github.com/dv-net/dv-processing/pkg/walletsdk/btc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/ltc"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/google/uuid"
)

type walletsServer struct {
	bs baseservices.IBaseServices

	walletv1connect.UnimplementedWalletServiceHandler
}

func newWalletsServer(
	bs baseservices.IBaseServices,
) *walletsServer {
	return &walletsServer{
		bs: bs,
	}
}

func (s *walletsServer) Name() string { return "wallets-server" }

func (s *walletsServer) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return walletv1connect.NewWalletServiceHandler(s, opts...)
}

func (s *walletsServer) GetOwnerHotWallets(
	ctx context.Context,
	request *connect.Request[walletv1.GetOwnerHotWalletsRequest],
) (*connect.Response[walletv1.GetOwnerHotWalletsResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	data, err := s.bs.Wallets().Hot().Find(ctx, wallets.FindHotWalletsParams{
		OwnerID:          &oid,
		Blockchain:       &blockchain,
		ExternalWalletID: request.Msg.ExternalWalletId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find hot wallets: %w", err))
	}

	addresses := make([]*walletv1.GetOwnerHotWalletsResponse_HotAddress, 0, len(data.Items))
	for _, wallet := range data.Items {
		addresses = append(addresses,
			&walletv1.GetOwnerHotWalletsResponse_HotAddress{
				Address:          wallet.Address,
				ExternalWalletId: wallet.ExternalWalletID,
			},
		)
	}

	return connect.NewResponse(&walletv1.GetOwnerHotWalletsResponse{Addresses: addresses}), nil
}

func (s *walletsServer) GetOwnerColdWallets(
	ctx context.Context,
	request *connect.Request[walletv1.GetOwnerColdWalletsRequest],
) (*connect.Response[walletv1.GetOwnerColdWalletsResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	params := wallets.FindColdWalletsParams{
		OwnerID: &oid,
	}

	if request.Msg.Blockchain != nil {
		blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.Blockchain = &blockchain
	}

	data, err := s.bs.Wallets().Cold().Find(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find cold wallets: %w", err))
	}

	items := make([]*walletv1.WalletPreview, 0, len(data.Items))
	for _, wallet := range data.Items {
		items = append(items, &walletv1.WalletPreview{
			Address:    wallet.Address,
			Blockchain: models.ConvertBlockchainTypeToPb(wallet.Blockchain),
		})
	}

	return connect.NewResponse(&walletv1.GetOwnerColdWalletsResponse{Items: items}), nil
}

func (s *walletsServer) GetOwnerProcessingWallets(
	ctx context.Context,
	request *connect.Request[walletv1.GetOwnerProcessingWalletsRequest],
) (*connect.Response[walletv1.GetOwnerProcessingWalletsResponse], error) {
	ctx = WithClientContext(ctx, *request)

	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		err = fmt.Errorf("owner id undefined: %w", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := wallets.FindProcessingWalletsParams{
		OwnerID: &oid,
	}

	if request.Msg.Blockchain != nil {
		blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.Blockchain = &blockchain
	}

	data, err := s.bs.Wallets().Processing().Find(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find processing wallets: %w", err))
	}

	items := make([]*walletv1.WalletPreview, 0, len(data.Items))
	for _, wallet := range data.Items {
		walletPreview := &walletv1.WalletPreview{
			Address:    wallet.Address,
			Blockchain: models.ConvertBlockchainTypeToPb(wallet.Blockchain),
			Assets:     new(walletv1.Assets),
		}

		if request.Msg.Tiny != nil && *request.Msg.Tiny {
			items = append(items, walletPreview)
			continue
		}

		assets, err := s.bs.EProxy().AddressBalances(ctx, wallet.Address, wallet.Blockchain)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get wallet balance: %w", err))
		}

		preparedAssets := &walletv1.Assets{Asset: make([]*walletv1.Asset, 0, len(assets))}
		for _, asset := range assets {
			preparedAssets.Asset = append(
				preparedAssets.Asset,
				&walletv1.Asset{
					Identity: asset.ID,
					Amount:   asset.Amount.String(),
				},
			)
		}
		walletPreview.Assets = preparedAssets

		walletPreview.BlockchainAdditionalData, err = s.walletBlockchainAdditionalData(ctx, wallet.Address, wallet.Blockchain)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get resources: %w", err))
		}

		items = append(items, walletPreview)
	}

	return connect.NewResponse(&walletv1.GetOwnerProcessingWalletsResponse{Items: items}), nil
}

func (s *walletsServer) CreateOwnerHotWallet(ctx context.Context, request *connect.Request[walletv1.CreateOwnerHotWalletRequest]) (*connect.Response[walletv1.CreateOwnerHotWalletResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	if request.Msg.GetExternalWalletId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("external wallet id undefined"))
	}

	blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	owner, err := s.bs.Owners().GetByID(ctx, oid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// find exists hot wallet
	findExistsWallet, err := s.bs.Wallets().Hot().Find(ctx, wallets.FindHotWalletsParams{
		OwnerID:          &owner.ID,
		Blockchain:       &blockchain,
		ExternalWalletID: &request.Msg.ExternalWalletId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find hot wallets: %w", err))
	}

	for _, wallet := range findExistsWallet.Items {
		if !wallet.IsDirty && wallet.IsActive {
			return connect.NewResponse(&walletv1.CreateOwnerHotWalletResponse{
				Address: wallet.Address,
			}), nil
		}
	}

	// create hot wallet if not exists
	var addressType string
	switch blockchain {
	case wconstants.BlockchainTypeBitcoin:
		addressType = convertBitcoinWalletType(request.Msg.GetBitcoinAddressType())
	case wconstants.BlockchainTypeLitecoin:
		addressType = convertLitecoinWalletType(request.Msg.GetLitecoinAddressType())
	}

	wallet, err := s.bs.Wallets().Hot().Create(ctx, wallets.CreateHotWalletParams{
		Blockchain:       blockchain,
		OwnerID:          owner.ID,
		ExternalWalletID: request.Msg.GetExternalWalletId(),
		Mnemonic:         owner.Mnemonic,
		Passphrase:       owner.PassPhrase.String,
		AddressType:      addressType,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create hot wallet: %w", err))
	}

	return connect.NewResponse(&walletv1.CreateOwnerHotWalletResponse{
		Address: wallet.Address,
	}), nil
}

func (s *walletsServer) MarkDirtyHotWallet(ctx context.Context, request *connect.Request[walletv1.MarkDirtyHotWalletRequest]) (*connect.Response[walletv1.MarkDirtyHotWalletResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	if request.Msg.GetAddress() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("address undefined"))
	}

	blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	owner, err := s.bs.Owners().GetByID(ctx, oid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.bs.Wallets().Hot().MarkDirty(ctx, owner.ID, blockchain, request.Msg.GetAddress()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mark dirty hot wallet: %w", err))
	}

	return connect.NewResponse(new(walletv1.MarkDirtyHotWalletResponse)), nil
}

func (s *walletsServer) AttachOwnerColdWallets(ctx context.Context, request *connect.Request[walletv1.AttachOwnerColdWalletsRequest]) (*connect.Response[walletv1.AttachOwnerColdWalletsResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	blockchain, err := models.ConvertBlockchainType(request.Msg.GetBlockchain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	owner, err := s.bs.Owners().GetByID(ctx, oid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if owner.OtpSecret.String == "" {
		return nil, fmt.Errorf("owner has no secret")
	}

	if !owner.OtpConfirmed {
		return nil, fmt.Errorf("two-factor authenticator is disabled")
	}

	if ok := totp.Validate(otp, owner.OtpSecret.String); !ok {
		return nil, fmt.Errorf("failed to validate totp")
	}

	batchParams := make([]wallets.CreateColdWalletParams, 0, len(request.Msg.GetAddresses()))
	for _, address := range request.Msg.GetAddresses() {
		params := wallets.CreateColdWalletParams{
			Blockchain: blockchain,
			Address:    address,
			OwnerID:    owner.ID,
		}

		batchParams = append(batchParams, params)
	}

	if err := s.bs.Wallets().Cold().BatchAttachColdWallets(ctx, owner.ID, blockchain, batchParams); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("batch create cold wallets: %w", err))
	}

	return connect.NewResponse(new(walletv1.AttachOwnerColdWalletsResponse)), nil
}

func (s *walletsServer) walletBlockchainAdditionalData(ctx context.Context, address string, blockchain wconstants.BlockchainType) (*walletv1.BlockchainAdditionalData, error) {
	addData := &walletv1.BlockchainAdditionalData{}

	if blockchain == wconstants.BlockchainTypeTron {
		defaultTronData := &walletv1.BlockchainAdditionalData_TronData{
			AvailableEnergyForUse:    "0",
			AvailableBandwidthForUse: "0",
			TotalEnergy:              "0",
			TotalBandwidth:           "0",
			StackedTrx:               "0",
			StackedBandwidth:         "0",
			StackedEnergy:            "0",
			TotalUsedEnergy:          "0",
			TotalUsedBandwidth:       "0",
		}

		trServ := s.bs.Tron()
		// blockchain is disabled by config
		if trServ == nil {
			addData.TronData = defaultTronData
			return addData, nil
		}

		res, err := trServ.AccountResourceInfo(ctx, address)
		if err != nil {
			if strings.Contains(err.Error(), "account not found") {
				addData.TronData = defaultTronData
				return addData, nil
			}

			return nil, fmt.Errorf("get tron resources: %w", err)
		}

		addData.TronData = &walletv1.BlockchainAdditionalData_TronData{
			AvailableEnergyForUse:    res.EnergyAvailableForUse.String(),
			AvailableBandwidthForUse: res.BandwidthAvailableForUse.String(),
			StackedTrx:               res.TotalStackedTRX.String(),
			StackedEnergy:            res.StackedEnergy.String(),
			StackedBandwidth:         res.StackedBandwidth.String(),
			StackedBandwidthTrx:      res.StackedBandwidthTRX.String(),
			StackedEnergyTrx:         res.StackedEnergyTRX.String(),
			TotalEnergy:              res.TotalEnergy.String(),
			TotalBandwidth:           res.TotalBandwidth.String(),
			TotalUsedEnergy:          res.TotalUsedEnergy.String(),
			TotalUsedBandwidth:       res.TotalUsedBandwidth.String(),
		}
	}

	return addData, nil
}

func convertBitcoinWalletType(wt commonv1.BitcoinAddressType) string {
	switch wt {
	case commonv1.BitcoinAddressType_BITCOIN_ADDRESS_TYPE_P2SH:
		return string(btc.AddressTypeP2SH)
	case commonv1.BitcoinAddressType_BITCOIN_ADDRESS_TYPE_P2PKH:
		return string(btc.AddressTypeP2PKH)
	case commonv1.BitcoinAddressType_BITCOIN_ADDRESS_TYPE_SEGWIT:
		return string(btc.AddressTypeP2WPKH)
	case commonv1.BitcoinAddressType_BITCOIN_ADDRESS_TYPE_P2TR:
		return string(btc.AddressTypeP2TR)
	default:
		return string(btc.AddressTypeP2TR)
	}
}

func convertLitecoinWalletType(wt commonv1.LitecoinAddressType) string {
	switch wt {
	case commonv1.LitecoinAddressType_LITECOIN_ADDRESS_TYPE_P2SH:
		return string(ltc.AddressTypeP2SH)
	case commonv1.LitecoinAddressType_LITECOIN_ADDRESS_TYPE_P2PKH:
		return string(ltc.AddressTypeP2PKH)
	case commonv1.LitecoinAddressType_LITECOIN_ADDRESS_TYPE_SEGWIT:
		return string(ltc.AddressTypeP2WPKH)
	case commonv1.LitecoinAddressType_LITECOIN_ADDRESS_TYPE_P2TR:
		return string(ltc.AddressTypeP2TR)
	default:
		return string(ltc.AddressTypeP2TR)
	}
}
