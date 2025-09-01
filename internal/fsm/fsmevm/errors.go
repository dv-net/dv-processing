package fsmevm

import (
	"errors"
	"fmt"
)

var errFailedTransfer = errors.New("failed transfer")

func newErrFailedTransfer(err error) error {
	return fmt.Errorf("%w: %w", errFailedTransfer, err)
}
