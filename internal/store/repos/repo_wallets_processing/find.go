package repo_wallets_processing

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
	OwnerID    *uuid.UUID
	Address    *string
	Blockchain *wconstants.BlockchainType
	storecmn.CommonFindParams
}

func (s *CustomQuerier) findBuilder(params FindParams, columns ...string) *sqlbuilder.SelectBuilder {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder()

	sb = sb.Select(columns...).
		From(TableNameProcessingWallets.String())

	if params.OwnerID != nil {
		sb.Where(sb.Equal(ColumnNameProcessingWalletsOwnerId.String(), params.OwnerID.String()))
	}

	if params.Address != nil {
		sb.Where(sb.Equal(ColumnNameProcessingWalletsAddress.String(), *params.Address))
	}

	if params.Blockchain != nil {
		sb.Where(sb.Equal(ColumnNameProcessingWalletsBlockchain.String(), params.Blockchain.String()))
	}

	return sb
}

func (s *CustomQuerier) Find(ctx context.Context, params FindParams) (*storecmn.FindResponse[*models.ProcessingWallet], error) {
	// init builder
	sb := s.findBuilder(params, ProcessingWalletsColumnNames().Strings()...)

	sb.OrderBy(ColumnNameProcessingWalletsCreatedAt.String()).Desc()

	// execute query
	var items []*models.ProcessingWallet
	sql, args := sb.Build()
	if err := pgxscan.Select(ctx, s.psql, &items, sql, args...); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	return &storecmn.FindResponse[*models.ProcessingWallet]{
		Items: items,
	}, nil
}
