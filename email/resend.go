package email

import (
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"
)

// Client sends transactional emails via Resend.
type Client struct {
	client    *resend.Client
	fromEmail string
	fromName  string
}

// NewClient returns a configured Resend client, or nil if not configured.
func NewClient(apiKey, fromEmail, fromName string) *Client {
	if apiKey == "" || fromEmail == "" {
		return nil
	}
	if fromName == "" {
		fromName = "VIPA Insurance"
	}
	return &Client{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// Send sends an email to the given address.
func (c *Client) Send(toEmail, toName, subject, htmlBody string) error {
	if c == nil {
		return fmt.Errorf("email: client not configured")
	}

	from := fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{toEmail},
		Subject: subject,
		Html:    htmlBody,
	}

	sent, err := c.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("email: resend send: %w", err)
	}

	log.Printf("Email sent to %s (%s): %s [id=%s]", toEmail, toName, subject, sent.Id)
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
