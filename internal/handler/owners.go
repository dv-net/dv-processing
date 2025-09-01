package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/samber/lo"

	ownerv1 "github.com/dv-net/dv-processing/api/processing/owner/v1"
	"github.com/dv-net/dv-processing/api/processing/owner/v1/ownerv1connect"
	"github.com/dv-net/dv-processing/internal/constants"
	"github.com/dv-net/dv-processing/internal/services/baseservices"
	"github.com/dv-net/dv-processing/internal/services/owners"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
)

type ownersServer struct {
	bs baseservices.IBaseServices

	ownerv1connect.UnimplementedOwnerServiceHandler
}

func newOwnersServer(
	bs baseservices.IBaseServices,
) *ownersServer {
	return &ownersServer{
		bs: bs,
	}
}

func (s *ownersServer) Name() string { return "owners-server" }

func (s *ownersServer) RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return ownerv1connect.NewOwnerServiceHandler(s, opts...)
}

func (s *ownersServer) Create(ctx context.Context, request *connect.Request[ownerv1.CreateRequest]) (*connect.Response[ownerv1.CreateResponse], error) {
	cid, err := uuid.Parse(request.Msg.GetClientId())
	if err != nil {
		err = fmt.Errorf("client id: %w", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	owner, err := s.bs.Owners().Create(ctx, owners.CreateParams{
		ClientID:   cid,
		ExternalID: request.Msg.GetExternalId(),
		Mnemonic:   request.Msg.GetMnemonic(),
	})
	if err != nil {
		if errors.Is(err, owners.ErrClientNotFound) || errors.Is(err, owners.ErrExternalIDExists) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("owner insert: %w", err))
	}

	return connect.NewResponse(&ownerv1.CreateResponse{Id: owner.ID.String()}), nil
}

func (s *ownersServer) GetSeeds(ctx context.Context, request *connect.Request[ownerv1.GetSeedsRequest]) (*connect.Response[ownerv1.GetSeedsResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		err = fmt.Errorf("owner id: %w", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	data, err := s.bs.Owners().GetSeeds(ctx, oid, otp)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ownerv1.GetSeedsResponse{
		Mnemonic:   data.Mnemonic,
		PassPhrase: data.PassPhrase,
	}), nil
}

func (s *ownersServer) GetPrivateKeys(
	ctx context.Context,
	request *connect.Request[ownerv1.GetPrivateKeysRequest],
) (*connect.Response[ownerv1.GetPrivateKeysResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	// Get all private keys for the owner.
	data, err := s.bs.Owners().GetAllPrivateKeys(ctx, owners.GetAllPrivateKeysRequest{
		OwnerID: oid,
		OTP:     otp,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &ownerv1.GetPrivateKeysResponse{
		Keys: make(map[string]*ownerv1.KeyPairSequence),
	}

	for blockchain, privateKeys := range *data {
		if res.Keys[blockchain.String()] == nil {
			res.Keys[blockchain.String()] = new(ownerv1.KeyPairSequence)
		}

		for _, privateKey := range privateKeys {
			res.Keys[blockchain.String()].Pairs = append(res.Keys[blockchain.String()].Pairs, &ownerv1.KeyPair{
				PublicKey:  privateKey.PublicKey,
				PrivateKey: privateKey.PrivateKey,
				Address:    privateKey.Address,
				Kind:       privateKey.Kind.String(),
			})
		}
	}

	return connect.NewResponse(res), nil
}

func (s *ownersServer) GetHotWalletKeys(ctx context.Context, request *connect.Request[ownerv1.GetHotWalletKeysRequest]) (*connect.Response[ownerv1.GetHotWalletKeysResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	otp := request.Msg.GetOtp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	data, err := s.bs.Owners().GetHotWalletKeys(ctx, &owners.GetHotWalletKeysRequest{
		OwnerID:           oid,
		OTP:               otp,
		WalletAddresses:   request.Msg.GetWalletAddresses(),
		ExcludedAddresses: request.Msg.GetExcludedWalletAddresses(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &ownerv1.GetHotWalletKeysResponse{
		Entries: make([]*ownerv1.GetHotWalletKeysItem, 0),
	}

	for blockchain, items := range *data {
		res.Entries = append(res.Entries, &ownerv1.GetHotWalletKeysItem{
			Name: constants.BlockchainTypeToPB(blockchain),
			Items: lo.Map(items, func(item owners.PrivateKeyItem, _ int) *ownerv1.PrivateKeyItem {
				return &ownerv1.PrivateKeyItem{
					Address:    item.Address,
					PublicKey:  item.PublicKey,
					PrivateKey: item.PrivateKey,
				}
			}),
		})
	}

	return connect.NewResponse(res), nil
}

// ConfirmTwoFactorAuth Confirm owner two auth
func (s *ownersServer) ConfirmTwoFactorAuth(
	ctx context.Context,
	request *connect.Request[ownerv1.ConfirmTwoFactorAuthRequest],
) (*connect.Response[ownerv1.ConfirmTwoFactorAuthResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	if err := s.bs.Owners().ConfirmTwoFactorAuth(ctx, oid, otp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("confirm two factor auth: %w", err))
	}

	return connect.NewResponse(new(ownerv1.ConfirmTwoFactorAuthResponse)), nil
}

// DisableTwoFactorAuth disables owner two-factor auth, removing secret.
func (s *ownersServer) DisableTwoFactorAuth(
	ctx context.Context,
	request *connect.Request[ownerv1.DisableTwoFactorAuthRequest],
) (*connect.Response[ownerv1.DisableTwoFactorAuthResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("owner id undefined: %w", err))
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("otp undefined"))
	}

	if err := s.bs.Owners().DisableTwoFactorAuth(ctx, oid, otp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("disable two factor auth: %w", err))
	}

	return connect.NewResponse(new(ownerv1.DisableTwoFactorAuthResponse)), nil
}

// GetTwoFactorAuthData returns two-factor auth data for the owner.
func (s *ownersServer) GetTwoFactorAuthData(ctx context.Context, request *connect.Request[ownerv1.GetTwoFactorAuthDataRequest]) (*connect.Response[ownerv1.GetTwoFactorAuthDataResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		err = fmt.Errorf("owner id undefined: %w", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	data, err := s.bs.Owners().GetTwoFactorAuthData(ctx, oid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ownerv1.GetTwoFactorAuthDataResponse{
		Secret:      data.Secret,
		IsConfirmed: data.IsConfirmed,
	}), nil
}

func (s *ownersServer) ValidateTwoFactorToken(
	ctx context.Context,
	request *connect.Request[ownerv1.ValidateTwoFactorTokenRequest],
) (*connect.Response[ownerv1.ValidateTwoFactorTokenResponse], error) {
	oid, err := uuid.Parse(request.Msg.GetOwnerId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, storecmn.ErrEmptyID)
	}

	otp := request.Msg.GetTotp()
	if otp == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, storecmn.ErrEmptyOTP)
	}

	// TODO: Decide on returning either error/true or false/true
	if err := s.bs.Owners().ValidateTwoFactorToken(ctx, oid, otp); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("validate two factor auth: %w", err))
	}

	return connect.NewResponse(new(ownerv1.ValidateTwoFactorTokenResponse)), nil
}
