package evm_test

import (
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetBaseFeeMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		baseFeeWei decimal.Decimal
		expected   decimal.Decimal
	}{
		{
			name:       "Base fee >= 200 GWei",
			baseFeeWei: decimal.NewFromInt(200000000000),
			expected:   decimal.NewFromFloat(1.14).Mul(decimal.NewFromInt(200000000000)),
		},
		{
			name:       "Base fee >= 100 GWei",
			baseFeeWei: decimal.NewFromInt(100000000000),
			expected:   decimal.NewFromFloat(1.17).Mul(decimal.NewFromInt(100000000000)),
		},
		{
			name:       "Base fee >= 40 GWei",
			baseFeeWei: decimal.NewFromInt(40000000000),
			expected:   decimal.NewFromFloat(1.18).Mul(decimal.NewFromInt(40000000000)),
		},
		{
			name:       "Base fee >= 20 GWei",
			baseFeeWei: decimal.NewFromInt(20000000000),
			expected:   decimal.NewFromFloat(1.19).Mul(decimal.NewFromInt(20000000000)),
		},
		{
			name:       "Base fee >= 10 GWei",
			baseFeeWei: decimal.NewFromInt(10000000000),
			expected:   decimal.NewFromFloat(1.192).Mul(decimal.NewFromInt(10000000000)),
		},
		{
			name:       "Base fee >= 9 GWei",
			baseFeeWei: decimal.NewFromInt(9000000000),
			expected:   decimal.NewFromFloat(1.195).Mul(decimal.NewFromInt(9000000000)),
		},
		{
			name:       "Base fee >= 8 GWei",
			baseFeeWei: decimal.NewFromInt(8000000000),
			expected:   decimal.NewFromFloat(1.20).Mul(decimal.NewFromInt(8000000000)),
		},
		{
			name:       "Base fee >= 7 GWei",
			baseFeeWei: decimal.NewFromInt(7000000000),
			expected:   decimal.NewFromFloat(1.215).Mul(decimal.NewFromInt(7000000000)),
		},
		{
			name:       "Base fee >= 6 GWei",
			baseFeeWei: decimal.NewFromInt(6000000000),
			expected:   decimal.NewFromFloat(1.22).Mul(decimal.NewFromInt(6000000000)),
		},
		{
			name:       "Base fee >= 5 GWei",
			baseFeeWei: decimal.NewFromInt(5000000000),
			expected:   decimal.NewFromFloat(1.24).Mul(decimal.NewFromInt(5000000000)),
		},
		{
			name:       "Base fee >= 4 GWei",
			baseFeeWei: decimal.NewFromInt(4000000000),
			expected:   decimal.NewFromFloat(1.26).Mul(decimal.NewFromInt(4000000000)),
		},
		{
			name:       "Base fee < 4 GWei",
			baseFeeWei: decimal.NewFromInt(3000000000),
			expected:   decimal.NewFromFloat(1.30).Mul(decimal.NewFromInt(3000000000)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evm.GetBaseFeeMultiplier(tt.baseFeeWei)
			fmt.Printf("base fee: %s, maxFeePerGas: %s\n", tt.baseFeeWei.String(), result.String())
			assert.True(t, tt.expected.Equal(result), "expected %s, got %s", tt.expected.String(), result.String())
		})
	}
}
