package evm

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type EtherUnit int

const (
	EtherUnitWei EtherUnit = iota
	EtherUnitKWei
	EtherUnitMWei
	EtherUnitGWei
	EtherUnitSzabo
	EtherUnitFinney
	EtherUnitEther
)

var UnitMap = map[EtherUnit]decimal.Decimal{
	EtherUnitWei:    decimal.NewFromInt(1),
	EtherUnitKWei:   decimal.NewFromInt(1e3),
	EtherUnitMWei:   decimal.NewFromInt(1e6),
	EtherUnitGWei:   decimal.NewFromInt(1e9),
	EtherUnitSzabo:  decimal.NewFromInt(1e12),
	EtherUnitFinney: decimal.NewFromInt(1e15),
	EtherUnitEther:  decimal.NewFromInt(1e18),
}

// Unit represents a currency value in the Ethereum blockchain.
// by default, the value is in Wei.
type Unit struct {
	value decimal.Decimal
}

// NewUnit creates a new currency value with the given value and unit.
func NewUnit(value decimal.Decimal, baseUnit EtherUnit) Unit {
	switch baseUnit {
	case EtherUnitWei:
		return Unit{value: value}
	case EtherUnitKWei:
		return Unit{value: value.Mul(UnitMap[EtherUnitKWei])}
	case EtherUnitMWei:
		return Unit{value: value.Mul(UnitMap[EtherUnitMWei])}
	case EtherUnitGWei:
		return Unit{value: value.Mul(UnitMap[EtherUnitGWei])}
	case EtherUnitSzabo:
		return Unit{value: value.Mul(UnitMap[EtherUnitSzabo])}
	case EtherUnitFinney:
		return Unit{value: value.Mul(UnitMap[EtherUnitFinney])}
	case EtherUnitEther:
		return Unit{value: value.Mul(UnitMap[EtherUnitEther])}
	default:
		return Unit{value: value}
	}
}

// NewUnitFromBigInt creates a new currency value with the given big integer value and unit.
func NewUnitFromBigInt(value *big.Int, unit EtherUnit) Unit {
	return NewUnit(decimal.NewFromBigInt(value, 0), unit)
}

// Value returns the currency value in the given unit.
func (c Unit) Value(unit EtherUnit) Unit {
	switch unit {
	case EtherUnitWei:
		return Unit{value: c.value}
	case EtherUnitKWei:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitKWei], 3)}
	case EtherUnitMWei:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitMWei], 6)}
	case EtherUnitGWei:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitGWei], 9)}
	case EtherUnitSzabo:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitSzabo], 12)}
	case EtherUnitFinney:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitFinney], 15)}
	case EtherUnitEther:
		return Unit{value: c.value.DivRound(UnitMap[EtherUnitEther], 18)}
	default:
		return Unit{value: c.value}
	}
}

// String returns the string representation of the currency value.
func (c Unit) String() string { return c.value.String() }

// Decimal returns the decimal representation of the currency value.
func (c Unit) Decimal() decimal.Decimal { return c.value }

// BigInt returns the big integer representation of the currency value.
func (c Unit) BigInt() *big.Int { return c.value.BigInt() }
