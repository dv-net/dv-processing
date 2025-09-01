package tron

import "github.com/shopspring/decimal"

/*

	TRX

*/

type TRX struct{ value decimal.Decimal }

func NewTRX(value decimal.Decimal) TRX { return TRX{value: value} }

// ToDecimal converts TRX to decimal.
func (t TRX) ToDecimal() decimal.Decimal { return t.value }

// ToSUN converts TRX decimal.Decimal SUN.
func (t TRX) ToSUN() decimal.Decimal {
	return t.value.Mul(decimal.NewFromInt(1e6))
}

// ToEnergy converts TRX to Energy.
func (t TRX) ToEnergy(energyFee int64) decimal.Decimal {
	return t.value.Div(decimal.NewFromInt(energyFee)).Mul(decimal.NewFromInt(1e6))
}

// ToBandwidth converts TRX to Bandwidth.
func (t TRX) ToBandwidth(transactionFee int64) decimal.Decimal {
	return t.value.Mul(decimal.NewFromInt(transactionFee))
}

/*

	Energy

*/

type Energy struct{ value decimal.Decimal }

func NewEnergy(value decimal.Decimal) Energy { return Energy{value: value} }

// ToDecimal converts Energy to decimal.
func (e Energy) ToDecimal() decimal.Decimal { return e.value }

// ToTRX converts Energy to TRX.
func (e Energy) ToTRX(energyFee int64) TRX {
	return NewTRX(e.value.Mul(decimal.NewFromInt(energyFee)).Div(decimal.NewFromInt(1e6)))
}

/*

	Bandwidth

*/

type Bandwidth struct{ value decimal.Decimal }

func NewBandwidth(value decimal.Decimal) Bandwidth { return Bandwidth{value: value} }

// ToDecimal converts Bandwidth to decimal.
func (b Bandwidth) ToDecimal() decimal.Decimal { return b.value }

// ToTRX converts Bandwidth to TRX.
func (b Bandwidth) ToTRX(transactionFee int64) TRX {
	return NewTRX(b.value.Div(decimal.NewFromInt(transactionFee)))
}
