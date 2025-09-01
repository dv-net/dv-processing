package evm_test

import (
	"math/big"
	"testing"

	"github.com/dv-net/dv-processing/pkg/walletsdk/evm"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUntis(t *testing.T) {
	tests := []struct {
		value    decimal.Decimal
		unit     evm.EtherUnit
		expected decimal.Decimal
	}{
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitKWei,
			expected: decimal.NewFromInt(1e3),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitMWei,
			expected: decimal.NewFromInt(1e6),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitGWei,
			expected: decimal.NewFromInt(1e9),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitSzabo,
			expected: decimal.NewFromInt(1e12),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitFinney,
			expected: decimal.NewFromInt(1e15),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitEther,
			expected: decimal.NewFromInt(1e18),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			unit := evm.NewUnit(test.value, test.unit)
			require.Equal(t, test.expected.String(), unit.String())
		})
	}
}

func TestFromBigInt(t *testing.T) {
	tests := []struct {
		value    *big.Int
		unit     evm.EtherUnit
		expected decimal.Decimal
	}{
		{
			value:    big.NewInt(1),
			unit:     evm.EtherUnitWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e3),
			unit:     evm.EtherUnitKWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e6),
			unit:     evm.EtherUnitMWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e9),
			unit:     evm.EtherUnitGWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e12),
			unit:     evm.EtherUnitSzabo,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e15),
			unit:     evm.EtherUnitFinney,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    big.NewInt(1e18),
			unit:     evm.EtherUnitEther,
			expected: decimal.NewFromInt(1),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			unit := evm.NewUnitFromBigInt(test.value, evm.EtherUnitWei)
			require.Equal(t, test.expected.String(), unit.Value(test.unit).String())
		})
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		value    decimal.Decimal
		unit     evm.EtherUnit
		expected decimal.Decimal
	}{
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitKWei,
			expected: decimal.NewFromInt(1e3),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitMWei,
			expected: decimal.NewFromInt(1e6),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitGWei,
			expected: decimal.NewFromInt(1e9),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitSzabo,
			expected: decimal.NewFromInt(1e12),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitFinney,
			expected: decimal.NewFromInt(1e15),
		},
		{
			value:    decimal.NewFromInt(1),
			unit:     evm.EtherUnitEther,
			expected: decimal.NewFromInt(1e18),
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			unit := evm.NewUnit(tc.value, tc.unit)
			require.Equal(t, tc.expected.String(), unit.Value(evm.EtherUnitWei).String())
		})
	}
}

func TestFromUnitToEqualUnit(t *testing.T) {
	tests := []struct {
		value    decimal.Decimal
		fromUnit evm.EtherUnit
		toUnit   evm.EtherUnit
		expected decimal.Decimal
	}{
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitWei,
			toUnit:   evm.EtherUnitWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitKWei,
			toUnit:   evm.EtherUnitKWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitMWei,
			toUnit:   evm.EtherUnitMWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitGWei,
			toUnit:   evm.EtherUnitGWei,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitSzabo,
			toUnit:   evm.EtherUnitSzabo,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitFinney,
			toUnit:   evm.EtherUnitFinney,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitEther,
			toUnit:   evm.EtherUnitEther,
			expected: decimal.NewFromInt(1),
		},
		{
			value:    decimal.NewFromInt(1),
			fromUnit: evm.EtherUnitSzabo,
			toUnit:   evm.EtherUnitGWei,
			expected: decimal.NewFromInt(1000),
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			unit := evm.NewUnit(tc.value, tc.fromUnit)
			require.Equal(t, tc.expected.String(), unit.Value(tc.toUnit).String())
		})
	}
}
