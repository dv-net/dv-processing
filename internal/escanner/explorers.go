package escanner

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/pkg/walletsdk/wconstants"
)

func (s *EScanner) initExplorers(ctx context.Context) error {
	if len(s.config.Available()) == 0 {
		return nil
	}

	errCh := make(chan error, 1)

	for _, chain := range s.config.Available() {
		go func() {
			if err := s.initExplorer(ctx, chain); err != nil {
				errCh <- fmt.Errorf("init explorer for %s: %w", chain.String(), err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *EScanner) initExplorer(ctx context.Context, blockchain wconstants.BlockchainType) error {
	sc := newScanner(
		s.logger,
		s.config,
		s.store,
		s.bs,
		s.tm,
		s.sdk,
		blockchain,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sc.start(ctx)
	}()

	if err := <-errCh; err != nil {
		return fmt.Errorf("start scanner for %s: %w", blockchain.String(), err)
	}

	return nil
}
