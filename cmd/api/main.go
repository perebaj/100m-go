package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	mock "github.com/perebaj/100m-go"

	"go.mercari.io/go-emv-code/mpm"
)

const (
	shouldCreatePurchaseIntention = true
	shouldCreateLimitReservation  = true
	shouldBillLimitReservation    = true
)

func main() {
	// create a cors middleware
	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			next.ServeHTTP(w, r)
		})
	}

	jsonContentType := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}

	// how to test it? curl http://localhost:4000/flamengo
	http.HandleFunc("/flamengo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "aqui é vasco porra")
	})

	//create a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "I'm alive 3! Trigger CI")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/pay", func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		log.Default().Printf("Request POST /pay received")
		var input PayInput
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		defer r.Body.Close()
		log.Default().Printf("Decoder input: %v", input)

		// Decode the bar code
		dst, err := mpm.Decode([]byte(input.BarCode))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		log.Default().Printf("Decoded bar code: %v", dst)

		// Create the pix data
		pixData, err := mock.NewPixData(dst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		log.Default().Printf("PixData: %v", pixData)

		// Create the purchase intention
		if shouldCreatePurchaseIntention {
			log.Default().Printf("Requesting purchase intention...")
			purchaseIntentionResp, err := mock.PurchaseIntention(mock.PurchaseIntentionPayload{
				SupplierCompanyID:      "f3ddbf3a-78ee-473f-88bc-67630eb1e901", // Demo stg supplier
				MerchantDocumentNumber: "85933514000152",                       // Demo stg merchant
				Email:                  "mock@email.com",                       // mock
				URLCallback:            "localhost:8080/v1/app-info",           // mock
				ExternalID:             "701a5756-1dca-433f-a484-36f753fa9b91", // mock
				AmountCents:            pixData.TransactionAmount,              // mock
				Contact: mock.Contact{
					Phone:   "119992453490", // mock
					Channel: "whatsapp",     // mock
				},
				Cart: mock.Cart{
					ShippingCents: pixData.TransactionAmount, // mock
					DiscountCents: 0,                         // mock
					Items:         []int{},                   // mock
				},
				Origin: "link", // mock
			})
			if err != nil {
				log.Default().Printf("Error: %v", err)
				w.WriteHeader(http.StatusFailedDependency)
				json.NewEncoder(w).Encode(err.Error())
				return
			}
			log.Default().Printf("PurchaseIntention suceeded: %+v", purchaseIntentionResp)

			if shouldCreateLimitReservation {
				// Create the limit reservation
				log.Default().Printf("Requesting limit reservation...")
				limitReservationResp, err := mock.LimitReservation(mock.LimitReservationPayload{
					IP:                  "186.192.133.17",         // mock
					PaymentOption:       []int{7},                 // mock
					PurchaseIntentionID: purchaseIntentionResp.ID, // mock
				})
				if err != nil {
					log.Default().Printf("Error: %v", err)
					w.WriteHeader(http.StatusFailedDependency)
					json.NewEncoder(w).Encode(err.Error())
					return
				}
				log.Default().Printf("LimitReservation suceeded: %+v", limitReservationResp)

				if shouldBillLimitReservation {
					// Bill the limit reservation
					log.Default().Printf("Billing limit reservation...")
					invoiceResp, err := mock.BillLimitReservation(limitReservationResp.ID, mock.BillLimitReservationPayload{
						ApprovedAt: time.Now().Format(time.RFC3339),
						Invoices: []mock.Invoice{
							{
								AmountCents: pixData.TransactionAmount, // mock
								ExternalID:  uuid.NewString(),          // mock
								NfeID:       input.BarCode,             // mock
								NfeNumber:   input.BarCode,             // mock
								Notes:       "Hackatino II",            // mock
							},
						},
						LastBatch: true,
					})
					if err != nil {
						log.Default().Printf("Error: %v", err)
						w.WriteHeader(http.StatusFailedDependency)
						json.NewEncoder(w).Encode(err.Error())
						return
					}
					log.Default().Printf("BillLimitReservation suceeded: %+v", invoiceResp)
				}
			}
		}

		output := &PayOutput{
			TransactionID: uuid.New().String(),
			PaymentType:   "pix",
			PixData:       *pixData,
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(output)
	})

	http.HandleFunc("/pay/boleto", func(w http.ResponseWriter, r *http.Request) {
		var input PayInput
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		defer r.Body.Close()
		boletoData := mock.NewBoletoData(input.BarCode)
		output := &PayOutput{
			TransactionID: uuid.New().String(),
			PaymentType:   "boleto",
			BoletoData:    *boletoData,
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(output)
	})

	port := os.Getenv("PORT")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		intPort = 4000 // default is 4000
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", intPort),
		Handler: cors(jsonContentType(http.DefaultServeMux)),
	}

	fmt.Printf("Server is running on port %d\n", intPort)
	srv.ListenAndServe()
}

type PayInput struct {
	BarCode string `json:"bar_code"`
}

type PayOutput struct {
	TransactionID string          `json:"transaction_id"`
	PaymentType   string          `json:"payment_type"`
	PixData       mock.PIXData    `json:"pix_data"`
	BoletoData    mock.BoletoData `json:"boleto_data"`
}
