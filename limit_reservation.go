package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var authToken = os.Getenv("AUTH_TOKEN")

// Payload struct to represent the JSON request payload
type LimitReservationPayload struct {
	IP                  string `json:"ip"`
	PaymentOption       []int  `json:"payment_option"`
	PurchaseIntentionID string `json:"purchase_intention_id"`
}

type LimitReservationResponse struct {
	ID string `json:"id"`
}

func LimitReservation(payload LimitReservationPayload) (*LimitReservationResponse, error) {
	url := "https://stg.picadinho.truepay.dev/v1/limit-reservations"

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

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d with response: %s", resp.StatusCode, string(body))
	}

	// Decode the response
	var respBody LimitReservationResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &respBody, nil
}
