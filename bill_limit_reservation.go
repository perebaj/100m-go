package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Invoice represents the structure of an invoice in the payload
type Invoice struct {
	AmountCents int    `json:"amount_cents"`
	ExternalID  string `json:"external_id"`
	NfeID       string `json:"nfe_id"`
	NfeNumber   string `json:"nfe_number"`
	Notes       string `json:"notes"`
}

// Payload represents the structure of the entire payload
type BillLimitReservationPayload struct {
	ApprovedAt string    `json:"approved_at"`
	Invoices   []Invoice `json:"invoices"`
	LastBatch  bool      `json:"last_batch"`
}

// Installment represents the structure of an installment in the payload
type Installment struct {
	ID             string `json:"id"`
	AmountCents    int    `json:"amount_cents"`
	SettlementDate string `json:"settlement_date"`
}

// ExpectedCashout represents the structure of an expected cashout in the payload
type ExpectedCashout struct {
	AmountCents    int    `json:"amount_cents"`
	SettlementDate string `json:"settlement_date"`
}

// Invoice represents the structure of an invoice in the payload
type InvoiceResponse struct {
	ID               string            `json:"id"`
	ShortID          string            `json:"short_id"`
	ExternalID       string            `json:"external_id"`
	AmountCents      int               `json:"amount_cents"`
	BuyerID          string            `json:"buyer_id"`
	BuyerCNPJ        string            `json:"buyer_cnpj"`
	Status           string            `json:"status"`
	Description      string            `json:"description"`
	CreatedAt        string            `json:"created_at"`
	Installments     []Installment     `json:"installments"`
	ExpectedCashouts []ExpectedCashout `json:"expected_cashouts"`
}

// Payload represents the structure of the entire payload
type Payload struct {
	Invoices []Invoice `json:"invoices"`
}

// BillLimitReservation sends a POST request with the provided payload to the specified URL
func BillLimitReservation(limitReservationId string, payload BillLimitReservationPayload) (*InvoiceResponse, error) {
	url := fmt.Sprintf("https://stg.picadinho.truepay.dev/v1/limit-reservations/%s/bill", limitReservationId)

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	// Create the HTTP client and perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d with response: %s", resp.StatusCode, string(body))
	}

	// Decode the response
	var respBody InvoiceResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &respBody, nil
}
