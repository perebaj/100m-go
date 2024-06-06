package mock

import (
	"fmt"
	"strconv"
	"strings"
)

type BoletoData struct {
	Value        string `json:"value"`
	BankCode     string `json:"bank_code"`
	CurrencyCode string `json:"currency_code"`
}

func NewBoletoData(barCode string) *BoletoData {
	value := barCode[37:47]
	value = strings.TrimLeft(value, "0")
	v, _ := strconv.Atoi(value)
	valueDivided := float64(v) / 100
	b := &BoletoData{
		BankCode:     barCode[:3],
		CurrencyCode: barCode[3:4],
		Value:        fmt.Sprintf("%.2f", valueDivided),
	}
	return b
}
