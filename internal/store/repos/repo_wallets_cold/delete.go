package repo_wallets_cold

import (
	"context"

	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type DeleteAllByOwnerIDParams struct {
	OwnerID    *uuid.UUID
	Blockchain *wconstants.BlockchainType
}

func (s *CustomQuerier) deleteBuilder(params DeleteAllByOwnerIDParams, table string) *sqlbuilder.DeleteBuilder {
	db := sqlbuilder.PostgreSQL.NewDeleteBuilder()

	db = db.DeleteFrom(table).
		Where(db.And(db.Equal(ColumnNameColdWalletsOwnerId.String(), params.OwnerID.String())), db.Equal(ColumnNameColdWalletsBlockchain.String(), params.Blockchain.String()))

	if params.OwnerID != nil {
		db.Where(db.Equal(ColumnNameColdWalletsOwnerId.String(), params.OwnerID.String()))
	}

	if params.Blockchain != nil {
		db.Where(db.Equal(ColumnNameColdWalletsBlockchain.String(), params.Blockchain.String()))
	}

	return db
}

func (s *CustomQuerier) Delete(ctx context.Context, params DeleteAllByOwnerIDParams) error {
	// init builder
	db := s.deleteBuilder(params, TableNameColdWallets.String())

	// execute query
	sql, args := db.Build()
	_, err := s.psql.Exec(ctx, sql, args...)
	return err
}
