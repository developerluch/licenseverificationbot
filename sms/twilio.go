package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// TwilioClient sends SMS messages via the Twilio REST API.
type TwilioClient struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

// NewTwilioClient creates a new Twilio SMS client. Returns nil if credentials are missing.
func NewTwilioClient(accountSID, authToken, fromNumber string) *TwilioClient {
	if accountSID == "" || authToken == "" || fromNumber == "" {
		return nil
	}
	return &TwilioClient{
		AccountSID: accountSID,
		AuthToken:  authToken,
		FromNumber: fromNumber,
	}
}

// twilioResponse represents the Twilio API response.
type twilioResponse struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// SendSMS sends a text message to the given phone number.
// The `to` number must be in E.164 format (e.g., +15551234567).
func (c *TwilioClient) SendSMS(to, body string) error {
	if c == nil {
		return fmt.Errorf("twilio: client not configured")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.AccountSID)

	formData := url.Values{
		"To":   {to},
		"From": {c.FromNumber},
		"Body": {body},
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("twilio: build request: %w", err)
	}

	req.SetBasicAuth(c.AccountSID, c.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var tr twilioResponse
		json.Unmarshal(respBody, &tr)
		return fmt.Errorf("twilio: HTTP %d: %s (code %d)", resp.StatusCode, tr.ErrorMessage, tr.ErrorCode)
	}

	var tr twilioResponse
	json.Unmarshal(respBody, &tr)
	log.Printf("SMS sent to %s (SID: %s, status: %s)", to, tr.SID, tr.Status)
	return nil
}
