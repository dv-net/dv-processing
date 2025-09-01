package repos

import (
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_cold"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_hot"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_wallets_processing"
	"github.com/dv-net/dv-processing/pkg/postgres"
)

type IWallets interface {
	Common(opts ...Option) repo_wallets.Querier
	Hot(opts ...Option) repo_wallets_hot.ICustomQuerier
	Cold(opts ...Option) repo_wallets_cold.ICustomQuerier
	Processing(opts ...Option) repo_wallets_processing.ICustomQuerier
}

type wallets struct {
	common     *repo_wallets.Queries
	hot        *repo_wallets_hot.CustomQuerier
	cold       *repo_wallets_cold.CustomQuerier
	processing *repo_wallets_processing.CustomQuerier
}

func newWalletsRepo(psql *postgres.Postgres) IWallets {
	return &wallets{
		common:     repo_wallets.New(psql.DB),
		hot:        repo_wallets_hot.NewCustom(psql.DB),
		cold:       repo_wallets_cold.NewCustom(psql.DB),
		processing: repo_wallets_processing.NewCustom(psql.DB),
	}
}

func (s *wallets) Common(opts ...Option) repo_wallets.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.common.WithTx(options.Tx)
	}

	return s.common
}

func (s *wallets) Hot(opts ...Option) repo_wallets_hot.ICustomQuerier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.hot.WithTx(options.Tx)
	}

	return s.hot
}

func (s *wallets) Cold(opts ...Option) repo_wallets_cold.ICustomQuerier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.cold.WithTx(options.Tx)
	}

	return s.cold
}

func (s *wallets) Processing(opts ...Option) repo_wallets_processing.ICustomQuerier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.processing.WithTx(options.Tx)
	}

	return s.processing
}
