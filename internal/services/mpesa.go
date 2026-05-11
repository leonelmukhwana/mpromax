package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MpesaService struct {
	Key         string
	Secret      string
	Passkey     string
	ShortCode   string
	CallbackURL string
}

type STKPushPayload struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

func (s *MpesaService) getAccessToken() (string, error) {
	// Use https://api.safaricom.co.ke/ for production
	url := "https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(s.Key, s.Secret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Safaricom: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode access token: %v", err)
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access_token not found in response: %v", result)
	}

	return token, nil
}

func (s *MpesaService) InitiateSTK(phone string, amount int, jobRef string) error {
	// 1. FORMAT PHONE NUMBER (Ensure 2547XXXXXXXX format)
	if len(phone) > 0 && phone[0] == '0' {
		phone = "254" + phone[1:]
	} else if len(phone) > 0 && phone[0] == '+' {
		phone = phone[1:]
	}

	token, err := s.getAccessToken()
	if err != nil {
		return fmt.Errorf("auth error: %v", err)
	}

	// 2. GENERATE TIMESTAMP IN EAT (East Africa Time)
	// Safaricom is strict about time synchronization
	loc, _ := time.LoadLocation("Africa/Nairobi")
	timestamp := time.Now().In(loc).Format("20060102150405")

	// 3. GENERATE PASSWORD
	password := base64.StdEncoding.EncodeToString([]byte(s.ShortCode + s.Passkey + timestamp))

	payload := STKPushPayload{
		BusinessShortCode: s.ShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phone,
		PartyB:            s.ShortCode,
		PhoneNumber:       phone,
		CallBackURL:       s.CallbackURL,
		AccountReference:  jobRef,
		TransactionDesc:   "Payment for " + jobRef,
	}

	body, _ := json.Marshal(payload)
	url := "https://sandbox.safaricom.co.ke/mpesa/stkpush/v1/processrequest"

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("DEBUG: Triggering STK for %s, Amount: %d, Time: %s\n", phone, amount, timestamp)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	// 4. PARSE FULL RESPONSE
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode mpesa response: %v", err)
	}

	// Safaricom debugging: Check terminal for these values
	fmt.Printf("DEBUG: Safaricom API Status: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG: Safaricom API Body: %+v\n", result)

	// A successful request MUST have ResponseCode "0"
	if result["ResponseCode"] != "0" {
		msg := result["errorMessage"]
		if msg == nil {
			msg = result["CustomerMessage"]
		}
		return fmt.Errorf("mpesa rejected request: %v", msg)
	}

	return nil
}
