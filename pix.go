package mock

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"go.mercari.io/go-emv-code/mpm"
)

type PIXData struct {
	PayloadFormatIndicator          string       `json:"PayloadFormatIndicator"`
	PointOfInitiationMethod         string       `json:"PointOfInitiationMethod"`
	MerchantAccountInformation      []TLVData    `json:"MerchantAccountInformation"`
	MerchantCategoryCode            string       `json:"MerchantCategoryCode"`
	TransactionCurrency             string       `json:"TransactionCurrency"`
	TransactionAmount               int          `json:"TransactionAmount"`
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
	transactionAmount, err := getTransactionAmount(dst)
	if err != nil {
		return nil, err
	}
	log.Default().Printf("----------------------------Transaction amount: %s", transactionAmount)

	transactionAmountInt, err := convertStringToInt(transactionAmount)
	if err != nil {
		return nil, err
	}

	p.TransactionAmount = transactionAmountInt
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

func convertStringToInt(value string) (int, error) {
	// Remove the decimal point from the string
	cleanedValue := strings.Replace(value, ".", "", 1)

	// Convert the cleaned string to an integer
	intValue, err := strconv.Atoi(cleanedValue)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string to integer: %v", err)
	}

	return intValue, nil
}

func getTransactionAmount(dst *mpm.Code) (string, error) {
	log.Default().Printf("Getting transaction amount...")

	if dst.TransactionAmount.String != "" {
		return dst.TransactionAmount.String, nil
	}

	pixInfoURL := ""
	log.Default().Printf("Transaction amount not found in the payload")

	log.Default().Printf("Checking MerchantAccountInformation for tag 25: %s", dst.MerchantAccountInformation[0].Value)
	value, err := getEMVValue(dst.MerchantAccountInformation[0].Value, "25")
	if err != nil {
		log.Default().Printf("Error getting EMV value: %v", err)
		return "", fmt.Errorf("error getting EMV value: %v", err)
	}
	log.Default().Printf("%s", value)
	pixInfoURL = value

	if pixInfoURL != "" {
		log.Default().Printf("Getting JWT code from %s...", pixInfoURL)
		jwtCode, err := requestPixJWTCode(fmt.Sprintf("https://%s", pixInfoURL))
		if err != nil {
			log.Default().Printf("Error getting JWT code: %v", err)
			return "", fmt.Errorf("error getting JWT code: %v", err)
		}
		log.Default().Printf("JWT code: %s", jwtCode)

		jwtPayload, err := ParseJWT(jwtCode)
		if err != nil {
			log.Default().Printf("Error parsing JWT: %v", err)
			return "", fmt.Errorf("error parsing JWT: %v", err)
		}
		log.Default().Printf("JWT payload: %v", jwtPayload)

		valueMap, ok := jwtPayload["valor"].(map[string]any)
		if !ok {
			log.Default().Printf("Error getting valor from JWT payload")
			return "", fmt.Errorf("error getting valor from JWT payload")
		}
		log.Default().Printf("valor: %v", valueMap)

		transactionAmount, ok := valueMap["original"].(string)
		if !ok {
			log.Default().Printf("Error getting original from valor")
			return "", fmt.Errorf("error getting original from valor")
		}
		log.Default().Printf("original.valor: %s", transactionAmount)

		return transactionAmount, nil
	}

	return "", fmt.Errorf("transaction amount not found")
}

func requestPixJWTCode(url string) (string, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Perform the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the Content-Type is application/jose
	if resp.Header.Get("Content-Type") != "application/jose" {
		return "", fmt.Errorf("unexpected Content-Type: %v", resp.Header.Get("Content-Type"))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Convert the response body to a string
	token := string(body)

	return token, nil
}

func getEMVValue(emv string, id string) (string, error) {
	if len(id) != 2 {
		return "", errors.New("ID must be a 2-digit string")
	}

	i := 0
	for i < len(emv) {
		if len(emv)-i < 4 {
			return "", errors.New("invalid EMV string format")
		}
		fieldID := emv[i : i+2]
		fieldLength, err := strconv.Atoi(emv[i+2 : i+4])
		if err != nil {
			return "", errors.New("invalid length format in EMV string")
		}

		if len(emv)-i < 4+fieldLength {
			return "", errors.New("invalid EMV string format")
		}

		fieldValue := emv[i+4 : i+4+fieldLength]

		if fieldID == id {
			return fieldValue, nil
		}

		i += 4 + fieldLength
	}

	return "", fmt.Errorf("ID %s not found in EMV string", id)
}

func ParseJWT(tokenString string) (map[string]any, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("unable to parse claims")
}
