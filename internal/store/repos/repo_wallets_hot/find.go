package repo_wallets_hot

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type FindParams struct {
	OwnerID          *uuid.UUID
	Blockchain       *wconstants.BlockchainType
	Address          *string
	ExternalWalletID *string
	IsDirty          *bool
}

func (s *CustomQuerier) findBuilder(params FindParams, columns ...string) *sqlbuilder.SelectBuilder {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder()

	sb = sb.Select(columns...).
		From(TableNameHotWallets.String())

	if params.OwnerID != nil {
		sb.Where(sb.Equal(ColumnNameHotWalletsOwnerId.String(), params.OwnerID.String()))
	}

	if params.Blockchain != nil {
		sb.Where(sb.Equal(ColumnNameHotWalletsBlockchain.String(), params.Blockchain.String()))
	}

	if params.Address != nil {
		sb.Where(sb.Equal(ColumnNameHotWalletsAddress.String(), params.Address))
	}

	if params.ExternalWalletID != nil {
		sb.Where(sb.Equal(ColumnNameHotWalletsExternalWalletId.String(), params.ExternalWalletID))
	}

	if params.IsDirty != nil {
		sb.Where(sb.Equal(ColumnNameHotWalletsIsDirty.String(), params.IsDirty))
	}

	return sb
}

func (s *CustomQuerier) Find(ctx context.Context, params FindParams) (*storecmn.FindResponse[*models.HotWallet], error) {
	// init builder
	sb := s.findBuilder(params, HotWalletsColumnNames().Strings()...)

	sb.OrderBy(ColumnNameHotWalletsCreatedAt.String()).Desc()

	// execute query
	var items []*models.HotWallet
	sql, args := sb.Build()
	if err := pgxscan.Select(ctx, s.psql, &items, sql, args...); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	return &storecmn.FindResponse[*models.HotWallet]{
		Items: items,
	}, nil
}
