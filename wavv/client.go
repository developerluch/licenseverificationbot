package wavv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL = "https://app4.wavv.com"
)

// Client talks to the WAVV v1 API.
type Client struct {
	http    *http.Client
	jwt     string // Bearer token for v1 API
	expires time.Time
}

// NewClient creates a WAVV API client with a pre-authenticated JWT.
func NewClient(jwt string, expiresAt time.Time) *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
		jwt:  jwt,
		expires: expiresAt,
	}
}

// IsExpired returns true if the JWT is expired or about to expire (within 1 day).
func (c *Client) IsExpired() bool {
	return time.Now().After(c.expires.Add(-24 * time.Hour))
}

// SetJWT replaces the JWT token.
func (c *Client) SetJWT(jwt string, expiresAt time.Time) {
	c.jwt = jwt
	c.expires = expiresAt
}

// JWT returns the current token (for storage/refresh).
func (c *Client) JWT() string {
	return c.jwt
}

// Authenticate obtains a v1 session JWT using a GHL auth token.
// The GHL auth token comes from: POST /api/ghl/v2/auth with a GHL session.
func Authenticate(ghlAuthToken string) (*Client, *AuthResponse, error) {
	body, _ := json.Marshal(map[string]string{"token": ghlAuthToken})

	req, err := http.NewRequest("POST", BaseURL+"/api/v1/auth", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("wavv auth: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://app4.wavv.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("wavv auth: request: %w", err)
	}
	defer resp.Body.Close()

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, nil, fmt.Errorf("wavv auth: decode: %w", err)
	}
	if !authResp.Success {
		return nil, &authResp, fmt.Errorf("wavv auth: failed: %s", authResp.Error)
	}

	// JWT has 30-day TTL
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	client := NewClient(authResp.JWT, expiresAt)
	return client, &authResp, nil
}

// AuthResponse is the response from /api/v1/auth.
type AuthResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	JWT     string `json:"jwt,omitempty"`
	User    *User  `json:"user,omitempty"`
}

// User is the WAVV user object from auth response.
type User struct {
	ID            string `json:"id"`
	VendorID      string `json:"vendorId"`
	VendorUserID  string `json:"vendorUserId"`
	TeamID        string `json:"teamId"`
	GroupID       string `json:"groupId"`
	SubgroupID    string `json:"subgroupId"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Email         string `json:"email"`
	NewUser       bool   `json:"newUser"`
	CRMName       string `json:"crmName"`
}

// TeamStats is the response from /api/v1/stats/team.
type TeamStats struct {
	Success bool                          `json:"success"`
	Error   string                        `json:"error,omitempty"`
	Usage   map[string]map[string]DayStat `json:"usage"` // teamMemberId -> date -> stats
}

// DayStat holds a single day's stats for one team member.
type DayStat struct {
	Calls         int                `json:"calls"`
	TalkTime      int                `json:"talktime"`      // seconds
	Conversations int                `json:"conversations"`
	ConvoGoalLen  int                `json:"convoGoalLength"` // seconds
	Dispositions  map[string]int     `json:"dispositions"`
	Outcomes      map[string]int     `json:"outcomes"`
	InboundCalls  int                `json:"inboundCalls"`
	InboundTalk   int                `json:"inboundTalktime"` // seconds
	InboundConvos int                `json:"inboundConversations"`
}

// TeamMember is from /api/v1/stats/team-members.
type TeamMember struct {
	UserID       string `json:"userId"`
	TeamMemberID string `json:"teamMemberId"`
	Name         string `json:"name"`
	Calls        int    `json:"calls"`
	Conversations int   `json:"conversations"`
	ContactsCalled int  `json:"contactsCalled"`
	TalkTime     int    `json:"talktime"`
}

// TeamMembersResponse is the response from /api/v1/stats/team-members.
type TeamMembersResponse struct {
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
	Members []TeamMember `json:"members"`
}

// GetTeamStats fetches team-wide stats for a date range.
func (c *Client) GetTeamStats(start, end string) (*TeamStats, error) {
	params := url.Values{}
	params.Set("start", start)
	params.Set("end", end)

	var result TeamStats
	if err := c.get("/api/v1/stats/team", params, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("wavv stats/team: %s", result.Error)
	}
	return &result, nil
}

// GetTeamMembers fetches team member list with summary stats.
func (c *Client) GetTeamMembers(start, end string) (*TeamMembersResponse, error) {
	params := url.Values{}
	params.Set("start", start)
	params.Set("end", end)

	var result TeamMembersResponse
	if err := c.get("/api/v1/stats/team-members", params, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("wavv stats/team-members: %s", result.Error)
	}
	return &result, nil
}

// get does a GET request with bearer auth.
func (c *Client) get(path string, params url.Values, dest interface{}) error {
	u := BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return fmt.Errorf("wavv GET %s: build: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("Origin", "https://app4.wavv.com")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("wavv GET %s: request: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("wavv GET %s: status %d: %s", path, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}
