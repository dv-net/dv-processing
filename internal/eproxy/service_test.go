package eproxy

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestIsTransientError(t *testing.T) {
	require.True(t, IsTransientError(connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded)))
	require.True(t, IsTransientError(connect.NewError(connect.CodeUnavailable, errors.New("down"))))
	require.True(t, IsTransientError(connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit"))))
	require.True(t, IsTransientError(fmt.Errorf("page 3: %w", connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded))))

	require.False(t, IsTransientError(nil))
	require.False(t, IsTransientError(connect.NewError(connect.CodeInvalidArgument, errors.New("bad"))))
	require.False(t, IsTransientError(errors.New("plain")))
}
