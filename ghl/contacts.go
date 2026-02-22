package ghl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

// Contact represents a GHL contact for upsert.
type Contact struct {
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	DiscordID    string
	Agency       string
	State        string
	Tags         []string
}

type upsertContactRequest struct {
	FirstName    string            `json:"firstName,omitempty"`
	LastName     string            `json:"lastName,omitempty"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	LocationID   string            `json:"locationId"`
	Tags         []string          `json:"tags,omitempty"`
	CustomFields []customFieldPair `json:"customFields,omitempty"`
}

type customFieldPair struct {
	ID    string `json:"id"`
	Value string `json:"field_value"`
}

type upsertContactResponse struct {
	Contact struct {
		ID string `json:"id"`
	} `json:"contact"`
	New bool `json:"new"`
}

// CreateOrUpdateContact upserts a contact in GHL. Returns the contact ID.
func (c *Client) CreateOrUpdateContact(ctx context.Context, contact Contact) (string, error) {
	req := upsertContactRequest{
		FirstName:  contact.FirstName,
		LastName:   contact.LastName,
		Email:      contact.Email,
		Phone:      contact.Phone,
		LocationID: c.locationID,
		Tags:       contact.Tags,
	}

	// Add custom fields
	if c.customFields.DiscordID != "" && contact.DiscordID != "" {
		req.CustomFields = append(req.CustomFields, customFieldPair{ID: c.customFields.DiscordID, Value: contact.DiscordID})
	}
	if c.customFields.Agency != "" && contact.Agency != "" {
		req.CustomFields = append(req.CustomFields, customFieldPair{ID: c.customFields.Agency, Value: contact.Agency})
	}
	if c.customFields.State != "" && contact.State != "" {
		req.CustomFields = append(req.CustomFields, customFieldPair{ID: c.customFields.State, Value: contact.State})
	}

	body, err := c.do(ctx, "POST", "/contacts/upsert", req)
	if err != nil {
		return "", err
	}

	var resp upsertContactResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("ghl: unmarshal contact response: %w", err)
	}

	if resp.New {
		log.Printf("GHL: created new contact %s for %s %s", resp.Contact.ID, contact.FirstName, contact.LastName)
	} else {
		log.Printf("GHL: updated contact %s for %s %s", resp.Contact.ID, contact.FirstName, contact.LastName)
	}

	return resp.Contact.ID, nil
}

// UpdateContactTags updates tags on a contact.
func (c *Client) UpdateContactTags(ctx context.Context, contactID string, tags []string) error {
	req := map[string]interface{}{
		"tags": tags,
	}
	_, err := c.do(ctx, "PUT", "/contacts/"+url.PathEscape(contactID), req)
	return err
}
