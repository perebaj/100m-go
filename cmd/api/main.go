package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.mercari.io/go-emv-code/mpm"
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
		fmt.Fprintf(w, "I'm alive 2!")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/pay", func(w http.ResponseWriter, r *http.Request) {
		var input PayInput
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		defer r.Body.Close()
		dst, err := mpm.Decode([]byte(input.BarCode))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		pixData, err := NewPixData(dst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(err.Error())
			return
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
		boletoData := NewBoletoData(input.BarCode)
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
	TransactionID string     `json:"transaction_id"`
	PaymentType   string     `json:"payment_type"`
	PixData       PIXData    `json:"pix_data"`
	BoletoData    BoletoData `json:"boleto_data"`
}

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

type PIXData struct {
	PayloadFormatIndicator          string       `json:"PayloadFormatIndicator"`
	PointOfInitiationMethod         string       `json:"PointOfInitiationMethod"`
	MerchantAccountInformation      []TLVData    `json:"MerchantAccountInformation"`
	MerchantCategoryCode            string       `json:"MerchantCategoryCode"`
	TransactionCurrency             string       `json:"TransactionCurrency"`
	TransactionAmount               string       `json:"TransactionAmount"`
	TipOrConvenienceIndicator       string       `json:"TipOrConvenienceIndicator"`
	ValueOfConvenienceFeeFixed      string       `json:"ValueOfConvenienceFeeFixed"`
	ValueOfConvenienceFeePercentage string       `json:"ValueOfConvenienceFeePercentage"`
	CountryCode                     string       `json:"CountryCode"`
	MerchantName                    string       `json:"MerchantName"`
	MerchantCity                    string       `json:"MerchantCity"`
	PostalCode                      string       `json:"PostalCode"`
	AdditionalDataFieldTemplate     string       `json:"AdditionalDataFieldTemplate"`
	MerchantInformation             MerchantData `json:"MerchantInformation"`
	UnreservedTemplates             []TLVData    `json:"UnreservedTemplates"`
}

type TLVData struct {
	Tag    string `json:"Tag"`
	Length string `json:"Length"`
	Value  string `json:"Value"`
}

type MerchantData struct {
	LanguagePreference string `json:"Language"`
	Name               string `json:"Name"`
	City               string `json:"City"`
	Valid              bool   `json:""`
}

func NewPixData(dst *mpm.Code) (*PIXData, error) {
	p := PIXData{}
	p.PayloadFormatIndicator = dst.PayloadFormatIndicator
	p.PointOfInitiationMethod = string(dst.PointOfInitiationMethod)
	var merchantAccountInformations []TLVData
	for _, v := range dst.MerchantAccountInformation {
		merchantAccountInformations = append(merchantAccountInformations, TLVData{
			Tag:    v.Tag,
			Length: v.Length,
			Value:  v.Value,
		})
	}
	p.MerchantAccountInformation = merchantAccountInformations
	p.MerchantCategoryCode = dst.MerchantCategoryCode
	p.TransactionCurrency = dst.TransactionCurrency
	p.TransactionAmount = dst.TransactionAmount.String
	tip, err := dst.TipOrConvenienceIndicator.Tokenize()
	if err != nil {
		return nil, err
	}
	p.TipOrConvenienceIndicator = tip
	p.ValueOfConvenienceFeeFixed = dst.ValueOfConvenienceFeeFixed.String
	p.ValueOfConvenienceFeePercentage = dst.ValueOfConvenienceFeePercentage.String
	p.CountryCode = dst.CountryCode
	p.MerchantName = dst.MerchantName
	p.MerchantCity = dst.MerchantCity
	p.PostalCode = dst.PostalCode
	p.AdditionalDataFieldTemplate = dst.AdditionalDataFieldTemplate
	p.MerchantInformation = MerchantData(dst.MerchantInformation)
	var unreservedTemplates []TLVData
	for _, v := range dst.UnreservedTemplates {
		unreservedTemplates = append(unreservedTemplates, TLVData{
			Tag:    v.Tag,
			Length: v.Length,
			Value:  v.Value,
		})
	}
	p.UnreservedTemplates = unreservedTemplates
	return &p, nil
}
