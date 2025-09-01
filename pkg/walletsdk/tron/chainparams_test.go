package tron_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainParams(t *testing.T) {
	tr, err := newTronSDK(tronNodeGRPCAddr)
	require.NoError(t, err)

	ctx := context.Background()

	tr.Start(ctx)
	defer tr.Stop(ctx)

	chainParams, err := tr.ChainParams(ctx)
	require.NoError(t, err)

	fmt.Printf("ChainParams: %+v\n", chainParams)
}
