package processedblocks

import (
	"context"
	"errors"
	"time"

	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/repos/repo_processed_blocks"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Create
func (s *Service) Create(ctx context.Context, blockchain wconstants.BlockchainType, number int64, hash string, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if number <= 0 {
		return errNumberLessOrEqualToZero
	}

	return s.store.ProcessedBlocks(opts...).Create(ctx, repo_processed_blocks.CreateParams{
		Blockchain: blockchain,
		Number:     number,
		Hash: pgtype.Text{
			String: hash,
			Valid:  true,
		},
	})
}

// // CreateWithHash
// func (s *Service) CreateWithHash(ctx context.Context, blockchain wconstants.BlockchainType, number int64, hash string, opts ...repos.Option) error {
// 	if blockchain == "" {
// 		return errBlockchainEmpty
// 	}

// 	if number <= 0 {
// 		return errNumberLessOrEqualToZero
// 	}

// 	return s.store.ProcessedBlocks(opts...).CreateWithHash(ctx, blockchain, number, hash)
// }

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

// UpdateNumberWithHash
func (s *Service) UpdateNumberWithHash(ctx context.Context, blockchain wconstants.BlockchainType, number int64, hash string, opts ...repos.Option) error {
	if blockchain == "" {
		return errBlockchainEmpty
	}

	if number <= 0 {
		return errNumberLessOrEqualToZero
	}

	return s.store.ProcessedBlocks(opts...).UpdateNumberWithHash(ctx, repo_processed_blocks.UpdateNumberWithHashParams{
		Blockchain: blockchain,
		Number:     number,
		Hash: pgtype.Text{
			String: hash,
			Valid:  true,
		},
	})
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

// LastBlock returns the last block info including hash
func (s *Service) LastBlock(ctx context.Context, blockchain wconstants.BlockchainType) (*ProcessedBlock, error) {
	if blockchain == "" {
		return nil, errBlockchainEmpty
	}

	data, err := s.store.ProcessedBlocks().LastBlock(ctx, blockchain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return &ProcessedBlock{
		Blockchain: data.Blockchain.String(),
		Number:     data.Number,
		Hash:       data.Hash.String,
		CreatedAt:  data.CreatedAt.Time,
		UpdatedAt:  &data.UpdatedAt.Time,
	}, nil
}

type ProcessedBlock struct {
	Blockchain string     `json:"blockchain"`
	Number     int64      `json:"number"`
	Hash       string     `json:"hash"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}
