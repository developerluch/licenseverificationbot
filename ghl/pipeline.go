package ghl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

// MoveToStage moves a contact's opportunity to a pipeline stage.
// botStage is the internal stage number (1-8), mapped via stageMap.
func (c *Client) MoveToStage(ctx context.Context, contactID string, botStage int) error {
	stageID, ok := c.stageMap[botStage]
	if !ok || stageID == "" {
		return nil // No mapping for this stage
	}

	req := map[string]interface{}{
		"stageId":    stageID,
		"pipelineId": c.pipelineID,
		"contactId":  contactID,
		"status":     "open",
	}

	// Search for existing opportunity
	searchPath := fmt.Sprintf("/opportunities/search?location_id=%s&pipeline_id=%s&contact_id=%s",
		url.QueryEscape(c.locationID), url.QueryEscape(c.pipelineID), url.QueryEscape(contactID))
	body, err := c.do(ctx, "GET", searchPath, nil)
	if err != nil {
		return c.createOpportunity(ctx, contactID, stageID)
	}

	type oppSearchResp struct {
		Opportunities []struct {
			ID string `json:"id"`
		} `json:"opportunities"`
	}
	var resp oppSearchResp
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Opportunities) == 0 {
		return c.createOpportunity(ctx, contactID, stageID)
	}

	oppID := resp.Opportunities[0].ID
	_, err = c.do(ctx, "PUT", "/opportunities/"+url.PathEscape(oppID), req)
	if err != nil {
		return fmt.Errorf("ghl: move stage: %w", err)
	}

	log.Printf("GHL: moved contact %s to stage %d (%s)", contactID, botStage, stageID)
	return nil
}

func (c *Client) createOpportunity(ctx context.Context, contactID, stageID string) error {
	req := map[string]interface{}{
		"pipelineId":      c.pipelineID,
		"pipelineStageId": stageID,
		"contactId":       contactID,
		"locationId":      c.locationID,
		"name":            "Agent Onboarding",
		"status":          "open",
	}

	_, err := c.do(ctx, "POST", "/opportunities/", req)
	if err != nil {
		return fmt.Errorf("ghl: create opportunity: %w", err)
	}
	log.Printf("GHL: created opportunity for contact %s at stage %s", contactID, stageID)
	return nil
}

// MarkOpportunityLost marks the contact's opportunity as lost (e.g., kicked).
func (c *Client) MarkOpportunityLost(ctx context.Context, contactID string) error {
	searchPath := fmt.Sprintf("/opportunities/search?location_id=%s&pipeline_id=%s&contact_id=%s",
		url.QueryEscape(c.locationID), url.QueryEscape(c.pipelineID), url.QueryEscape(contactID))
	body, err := c.do(ctx, "GET", searchPath, nil)
	if err != nil {
		return nil
	}

	type oppSearchResp struct {
		Opportunities []struct {
			ID string `json:"id"`
		} `json:"opportunities"`
	}
	var resp oppSearchResp
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Opportunities) == 0 {
		return nil
	}

	req := map[string]interface{}{
		"status": "lost",
	}
	_, err = c.do(ctx, "PUT", "/opportunities/"+url.PathEscape(resp.Opportunities[0].ID), req)
	return err
}
