package repo_transfers

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type FindParams struct {
	OwnerID       *uuid.UUID
	StatusesIn    []string
	StatusesNotIn []string
	FromAddress   *string
	ToAddress     *string
	Blockchain    *wconstants.BlockchainType
	Limit         *int
}

func (s *CustomQuerier) findCTEBuilder(params FindParams, columns ...string) *sqlbuilder.SelectBuilder {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder()

	sb = sb.Select(columns...).
		From(TableNameTransfers.String())

	if len(params.StatusesIn) > 0 {
		sb.Where(sb.In(ColumnNameTransfersStatus.String(), util.ConvertListToAny(params.StatusesIn)...))
	}

	if len(params.StatusesNotIn) > 0 {
		sb.Where(sb.NotIn(ColumnNameTransfersStatus.String(), util.ConvertListToAny(params.StatusesNotIn)...))
	}

	if params.OwnerID != nil {
		sb.Where(sb.Equal(ColumnNameTransfersOwnerId.String(), params.OwnerID.String()))
	}

	if params.Blockchain != nil {
		sb.Where(sb.Equal(ColumnNameTransfersBlockchain.String(), params.Blockchain.String()))
	}

	if params.Limit != nil && *params.Limit > 0 {
		sb.Limit(*params.Limit)
	}

	return sb
}

func (s *CustomQuerier) Find(ctx context.Context, params FindParams) ([]*models.Transfer, error) {
	// init builder
	sb := sqlbuilder.With(
		sqlbuilder.CTETable("transfersCTE").As(
			s.findCTEBuilder(params, TransfersColumnNames().Strings()...),
		),
	).
		Select(TransfersColumnNames().Strings()...).
		OrderBy(ColumnNameTransfersCreatedAt.String()).
		Desc()

	if params.FromAddress != nil && *params.FromAddress != "" {
		sb.Where(
			fmt.Sprintf("%s @> ARRAY['%s']::varchar[]", ColumnNameTransfersFromAddresses.String(), *params.FromAddress),
		)
	}

	if params.ToAddress != nil && *params.ToAddress != "" {
		sb.Where(
			fmt.Sprintf("%s @> ARRAY['%s']::varchar[]", ColumnNameTransfersToAddresses.String(), *params.ToAddress),
		)
	}

	sb.SetFlavor(sqlbuilder.PostgreSQL)

	// execute query
	var items []*models.Transfer
	sql, args := sb.Build()
	if err := pgxscan.Select(ctx, s.psql, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}
