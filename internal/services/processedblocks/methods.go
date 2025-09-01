package processedblocks

import (
	"context"
	"errors"

	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/jackc/pgx/v5"
)

// Create
func (s *Service) Create(ctx context.Context, blockchain wconstants.BlockchainType, number int64, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if number <= 0 {
		return errNumberLessOrEqualToZero
	}

	return s.store.ProcessedBlocks(opts...).Create(ctx, blockchain, number)
}

// UpdateNumber
func (s *Service) UpdateNumber(ctx context.Context, blockchain wconstants.BlockchainType, number int64, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if number <= 0 {
		return errNumberLessOrEqualToZero
	}

	return s.store.ProcessedBlocks(opts...).UpdateNumber(ctx, blockchain, number)
}

// LastBlockNumber
func (s *Service) LastBlockNumber(ctx context.Context, blockchain wconstants.BlockchainType) (int64, error) {
	if blockchain == "" {
		return 0, errBlockchainEmpty
	}

	data, err := s.store.ProcessedBlocks().LastBlockNumber(ctx, blockchain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storecmn.ErrNotFound
		}
		return 0, err
	}

	return data, nil
}
