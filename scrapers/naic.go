package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// SessionFactory creates a new TLS client session with isolated cookies.
type SessionFactory func() (tls_client.HttpClient, error)

// naicAPIResponse matches the JSON structure from the NAIC SBS API.
type naicAPIResponse struct {
	Name                  string      `json:"name"`
	NPN                   string      `json:"npn"`
	LicenseNumber         json.Number `json:"licenseNumber"`
	LicenseType           string      `json:"licenseType"`
	LicenseTypeCode       string      `json:"licenseTypeCode"`
	LicenseEffectiveDate  string      `json:"licenseEffectiveDate"`
	LicenseExpirationDate string      `json:"licenseExpirationDate"`
	LOAs                  string      `json:"loas"`
	Residency             string      `json:"residency"`
	BusinessAddress       string      `json:"businessAddress"`
	BusinessPhone         string      `json:"businessPhone"`
}

// naicErrorResponse matches the NAIC error JSON.
type naicErrorResponse struct {
	UnexpectedError string `json:"UnexpectedError"`
}

// NAICScraper queries the NAIC SBS API for license information.
type NAICScraper struct {
	stateCode      string
	sessionFactory SessionFactory
}

// NewNAICScraper creates a new NAIC scraper for the given state code.
func NewNAICScraper(sessionFactory SessionFactory, stateCode string) *NAICScraper {
	return &NAICScraper{
		stateCode:      strings.ToUpper(stateCode),
		sessionFactory: sessionFactory,
	}
}

// StateCode returns the two-letter state code this scraper handles.
func (s *NAICScraper) StateCode() string { return s.stateCode }

// ManualLookupURL returns the URL for manual license lookups on the NAIC SBS portal.
func (s *NAICScraper) ManualLookupURL() string {
	return "https://sbs.naic.org/solar/external/pages/#/search/licensee/search"
}

// LookupByName searches for licenses by first and last name.
func (s *NAICScraper) LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error) {
	params := url.Values{
		"jurisdiction": {s.stateCode},
		"searchType":   {"Licensee"},
		"entityType":   {"IND"},
		"firstName":    {firstName},
		"lastName":     {lastName},
	}
	return s.query(ctx, params)
}

// LookupByNPN searches for licenses by National Producer Number.
func (s *NAICScraper) LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error) {
	params := url.Values{
		"jurisdiction": {s.stateCode},
		"searchType":   {"Licensee"},
		"entityType":   {"IND"},
		"npn":          {npn},
	}
	return s.query(ctx, params)
}

// LookupByLicenseNumber searches for licenses by license number.
func (s *NAICScraper) LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error) {
	params := url.Values{
		"jurisdiction":  {s.stateCode},
		"searchType":    {"Licensee"},
		"entityType":    {"IND"},
		"licenseNumber": {licenseNumber},
	}
	return s.query(ctx, params)
}

// query performs the actual HTTP request to the NAIC API and parses the response.
func (s *NAICScraper) query(ctx context.Context, params url.Values) ([]LicenseResult, error) {
	session, err := s.sessionFactory()
	if err != nil {
		return nil, fmt.Errorf("naic: session error: %w", err)
	}

	reqURL := "https://services.naic.org/api/licenseLookup/search?" + params.Encode()
	log.Printf("NAIC query: %s", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("naic: request error: %w", err)
	}

	req.Header = http.Header{
		"Accept":            {"application/json"},
		"Origin":            {"https://sbs.naic.org"},
		"Referer":           {"https://sbs.naic.org/"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		http.HeaderOrderKey: {"Accept", "Origin", "Referer", "Accept-Language"},
	}

	resp, err := session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("naic: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("naic: read body failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("naic: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Check for error response
	var errResp naicErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.UnexpectedError != "" {
		return nil, fmt.Errorf("naic: API error: %s", errResp.UnexpectedError)
	}

	// Parse as array of results
	var apiResults []naicAPIResponse
	if err := json.Unmarshal(body, &apiResults); err != nil {
		// Might be empty or unexpected format
		return []LicenseResult{{Found: false, State: s.stateCode}}, nil
	}

	if len(apiResults) == 0 {
		return []LicenseResult{{Found: false, State: s.stateCode}}, nil
	}

	var results []LicenseResult
	for _, r := range apiResults {
		// Parse active status from license type string
		active := strings.Contains(strings.ToLower(r.LicenseType), "active")

		// Parse residency
		resident := strings.EqualFold(r.Residency, "Yes")

		// Parse LOAs - replace <br/> with newline for readability
		loas := strings.ReplaceAll(r.LOAs, "<br/>", "\n")
		loas = strings.ReplaceAll(loas, "&lt;br/&gt;", "\n")

		// Parse name (format: "LAST, FIRST")
		fullName := r.Name

		results = append(results, LicenseResult{
			Found:          true,
			Active:         active,
			Resident:       resident,
			FullName:       fullName,
			LicenseNumber:  r.LicenseNumber.String(),
			NPN:            r.NPN,
			State:          s.stateCode,
			LicenseType:    r.LicenseType,
			Status:         r.LicenseType, // NAIC doesn't have separate status, it's embedded in type
			ExpirationDate: r.LicenseExpirationDate,
			IssueDate:      r.LicenseEffectiveDate,
			LOAs:           loas,
		})
	}

	return results, nil
}
