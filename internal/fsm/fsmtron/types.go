package fsmtron

import "github.com/shopspring/decimal"

type delegateStateData struct {
	TxHash          string          `json:"tx_hash"`
	FromAddress     string          `json:"from_address"`
	ToAddress       string          `json:"to_address"`
	Amount          decimal.Decimal `json:"amount"`
	Coeff           decimal.Decimal `json:"coeff"`
	AmountWithCoeff decimal.Decimal `json:"amount_with_coeff"`
	AmountInTrx     int64           `json:"amount_in_trx"`
}

type reclaimStateData struct {
	TxHash      string `json:"tx_hash"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Amount      int64  `json:"amount"`
}
