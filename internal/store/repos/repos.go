package repos

import (
	"github.com/dv-net/dv-processing/internal/store/repos/repo_clients"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_owners"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_processed_blocks"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_processed_incidents"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_settings"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_system"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfer_transactions"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_transfers"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_webhooks"
	"github.com/dv-net/dv-processing/pkg/postgres"
)

type IRepos interface {
	ProcessedBlocks(opts ...Option) repo_processed_blocks.Querier
	ProcessedIncidents(opts ...Option) repo_processed_incidents.Querier
	Clients(opts ...Option) repo_clients.Querier
	Owners(opts ...Option) repo_owners.Querier
	Transfers(opts ...Option) repo_transfers.ICustomQuerier
	Webhooks(opts ...Option) repo_webhooks.Querier
	Settings(opts ...Option) repo_settings.Querier
	TransferTransactions(opts ...Option) repo_transfer_transactions.Querier
	System() repo_system.ICustomQuerier
	Wallets() IWallets
}

type repos struct {
	processedBlocks      *repo_processed_blocks.Queries
	processedIncidents   *repo_processed_incidents.Queries
	clients              *repo_clients.Queries
	owners               *repo_owners.Queries
	transfers            *repo_transfers.CustomQuerier
	webhooks             *repo_webhooks.Queries
	settings             *repo_settings.Queries
	transferTransactions *repo_transfer_transactions.Queries
	system               *repo_system.CustomQuerier
	wallets              IWallets
}

func New(psql *postgres.Postgres) IRepos {
	return &repos{
		processedBlocks:      repo_processed_blocks.New(psql.DB),
		processedIncidents:   repo_processed_incidents.New(psql.DB),
		clients:              repo_clients.New(psql.DB),
		owners:               repo_owners.New(psql.DB),
		transfers:            repo_transfers.NewCustom(psql.DB),
		webhooks:             repo_webhooks.New(psql.DB),
		settings:             repo_settings.New(psql.DB),
		transferTransactions: repo_transfer_transactions.New(psql.DB),
		system:               repo_system.NewCustom(psql.DB),
		wallets:              newWalletsRepo(psql),
	}
}

// ProcessedBlocks
func (s *repos) ProcessedBlocks(opts ...Option) repo_processed_blocks.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.processedBlocks.WithTx(options.Tx)
	}

	return s.processedBlocks
}

// ProcessedIncidents
func (s *repos) ProcessedIncidents(opts ...Option) repo_processed_incidents.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.processedIncidents.WithTx(options.Tx)
	}

	return s.processedIncidents
}

// Clients
func (s *repos) Clients(opts ...Option) repo_clients.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.clients.WithTx(options.Tx)
	}

	return s.clients
}

// Owners
func (s *repos) Owners(opts ...Option) repo_owners.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.owners.WithTx(options.Tx)
	}

	return s.owners
}

// Transfers
func (s *repos) Transfers(opts ...Option) repo_transfers.ICustomQuerier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.transfers.WithTx(options.Tx)
	}

	return s.transfers
}

// Webhooks
func (s *repos) Webhooks(opts ...Option) repo_webhooks.Querier {
	options := parseOptions(opts...)

	if options.Tx != nil {
		return s.webhooks.WithTx(options.Tx)
	}

	return s.webhooks
}

// Settings
func (s *repos) Settings(opts ...Option) repo_settings.Querier {
	option := parseOptions(opts...)

	if option.Tx != nil {
		return s.settings.WithTx(option.Tx)
	}

	return s.settings
}

// TransferTransactions
func (s *repos) TransferTransactions(opts ...Option) repo_transfer_transactions.Querier {
	options := parseOptions(opts...)
	if options.Tx != nil {
		return s.transferTransactions.WithTx(options.Tx)
	}

	return s.transferTransactions
}

// System
func (s *repos) System() repo_system.ICustomQuerier {
	return s.system
}

// Wallets
func (s *repos) Wallets() IWallets {
	return s.wallets
}
