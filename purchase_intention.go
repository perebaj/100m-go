package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Contact struct {
	Phone   string `json:"phone"`
	Channel string `json:"channel"`
}

type Cart struct {
	ShippingCents int   `json:"shipping_cents"`
	DiscountCents int   `json:"discount_cents"`
	Items         []int `json:"items"`
}

type PurchaseIntentionPayload struct {
	SupplierCompanyID      string  `json:"supplier_company_id"`
	MerchantDocumentNumber string  `json:"merchant_document_number"`
	Email                  string  `json:"email"`
	URLCallback            string  `json:"url_callback"`
	ExternalID             string  `json:"external_id"`
	AmountCents            int     `json:"amount_cents"`
	Contact                Contact `json:"contact"`
	Cart                   Cart    `json:"cart"`
	Origin                 string  `json:"origin"`
}

type PurchaseIntentionResponse struct {
	ID string `json:"id"`
}

func PurchaseIntention(payload PurchaseIntentionPayload) (*PurchaseIntentionResponse, error) {
	url := "https://stg.picadinho.truepay.dev/v1/purchase-intentions"

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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	// Create the HTTP client and perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d with response: %s", resp.StatusCode, string(body))
	}

	var responseBody PurchaseIntentionResponse
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &responseBody, nil
}
