package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Client sends transactional emails via SendGrid's v3 API.
type Client struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// NewClient returns a configured SendGrid client, or nil if not configured.
func NewClient(apiKey, fromEmail, fromName string) *Client {
	if apiKey == "" || fromEmail == "" {
		return nil
	}
	if fromName == "" {
		fromName = "VIPA Insurance"
	}
	return &Client{
		APIKey:    apiKey,
		FromEmail: fromEmail,
		FromName:  fromName,
	}
}

type sgMail struct {
	Personalizations []sgPersonalization `json:"personalizations"`
	From             sgAddress           `json:"from"`
	Subject          string              `json:"subject"`
	Content          []sgContent         `json:"content"`
}

type sgPersonalization struct {
	To []sgAddress `json:"to"`
}

type sgAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Send sends an email to the given address.
func (c *Client) Send(toEmail, toName, subject, htmlBody string) error {
	if c == nil {
		return fmt.Errorf("email: client not configured")
	}

	payload := sgMail{
		Personalizations: []sgPersonalization{
			{To: []sgAddress{{Email: toEmail, Name: toName}}},
		},
		From:    sgAddress{Email: c.FromEmail, Name: c.FromName},
		Subject: subject,
		Content: []sgContent{
			{Type: "text/html", Value: htmlBody},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Email sent to %s (%s): %s", toEmail, toName, subject)
	return nil
}

// SendReminder sends a license verification reminder email.
func (c *Client) SendReminder(toEmail, toName string, daysLeft int) error {
	subject := fmt.Sprintf("VIPA: %d days left to verify your license", daysLeft)

	urgencyColor := "#3498db" // blue
	if daysLeft <= 7 {
		urgencyColor = "#e74c3c" // red
	} else if daysLeft <= 14 {
		urgencyColor = "#f39c12" // orange
	}

	html := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
  <div style="background: %s; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0;">
    <h1 style="margin: 0;">License Verification Reminder</h1>
  </div>
  <div style="padding: 20px; background: #f9f9f9; border-radius: 0 0 8px 8px;">
    <p>Hi <strong>%s</strong>,</p>
    <p>You have <strong>%d days</strong> remaining to verify your insurance license with VIPA.</p>
    <p>To verify, open Discord and use the command:</p>
    <pre style="background: #eee; padding: 10px; border-radius: 4px;">/verify first_name:YourFirst last_name:YourLast state:XX</pre>
    <p>If you need help, contact your upline or reply to this email.</p>
    <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
    <p style="color: #999; font-size: 12px;">You're receiving this because you opted in to email notifications during VIPA onboarding.
    To stop receiving these emails, use <code>/email-optout</code> in Discord.</p>
  </div>
</div>`, urgencyColor, toName, daysLeft)

	return c.Send(toEmail, toName, subject, html)
}

// SendDeadlineExpired notifies the user their deadline has passed.
func (c *Client) SendDeadlineExpired(toEmail, toName string) error {
	subject := "VIPA: Your license verification deadline has passed"

	html := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
  <div style="background: #e74c3c; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0;">
    <h1 style="margin: 0;">Verification Deadline Expired</h1>
  </div>
  <div style="padding: 20px; background: #f9f9f9; border-radius: 0 0 8px 8px;">
    <p>Hi <strong>%s</strong>,</p>
    <p>Your 30-day license verification deadline has passed. An admin has been notified.</p>
    <p>Please contact your upline or check the VIPA Discord server for next steps.</p>
    <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
    <p style="color: #999; font-size: 12px;">You're receiving this because you opted in to email notifications during VIPA onboarding.</p>
  </div>
</div>`, toName)

	return c.Send(toEmail, toName, subject, html)
}

// SendVerificationSuccess notifies the user their license was verified.
func (c *Client) SendVerificationSuccess(toEmail, toName, state, licenseNum string) error {
	subject := "VIPA: Your license has been verified!"

	html := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
  <div style="background: #2ecc71; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0;">
    <h1 style="margin: 0;">License Verified!</h1>
  </div>
  <div style="padding: 20px; background: #f9f9f9; border-radius: 0 0 8px 8px;">
    <p>Hi <strong>%s</strong>,</p>
    <p>Great news! Your insurance license has been verified.</p>
    <ul>
      <li><strong>State:</strong> %s</li>
      <li><strong>License #:</strong> %s</li>
    </ul>
    <p>You've been promoted to <strong>Licensed Agent</strong> on the VIPA Discord server.</p>
    <p>Next step: Use <code>/contract</code> in Discord to book your contracting appointment.</p>
    <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
    <p style="color: #999; font-size: 12px;">You're receiving this because you opted in to email notifications during VIPA onboarding.</p>
  </div>
</div>`, toName, state, licenseNum)

	return c.Send(toEmail, toName, subject, html)
}
