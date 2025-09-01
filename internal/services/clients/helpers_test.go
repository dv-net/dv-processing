package clients

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDomain(t *testing.T) {
	t.Run("valid domain", func(t *testing.T) {
		domain := "example.com"
		require.True(t, validateDomain(context.Background(), domain))
	})
	t.Run("invalid domain", func(t *testing.T) {
		domain := "1.com"
		require.False(t, validateDomain(context.Background(), domain))
	})
}
