package scrapers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/PuerkitoBio/goquery"
	"license-bot-go/scrapers/captcha"
)

const caBaseURL = "https://cdicloud.insurance.ca.gov/cal"
// TODO: Extract Turnstile site key from page HTML dynamically instead of hardcoding.
// If CA DOI rotates the key, this will need updating.
const caTurnstileSiteKey = "0x4AAAAAAAeV7o-X_350Kljk"

type CaliforniaScraper struct {
	sessionFactory SessionFactory
	capSolver      *captcha.CapSolver
}

func NewCaliforniaScraper(sessionFactory SessionFactory, cs *captcha.CapSolver) *CaliforniaScraper {
	return &CaliforniaScraper{sessionFactory: sessionFactory, capSolver: cs}
}

func (s *CaliforniaScraper) StateCode() string      { return "CA" }
func (s *CaliforniaScraper) ManualLookupURL() string { return "https://cdicloud.insurance.ca.gov/cal/" }

func (s *CaliforniaScraper) LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error) {
	if s.capSolver == nil {
		return []LicenseResult{{
			Found: false,
			State: "CA",
			Error: "CA DOI requires CAPTCHA solving. CAPSOLVER_API_KEY not configured. Verify manually at " + s.ManualLookupURL(),
		}}, nil
	}

	log.Printf("CA lookup: %s %s", firstName, lastName)

	session, err := s.sessionFactory()
	if err != nil {
		return nil, fmt.Errorf("ca: session error: %w", err)
	}

	// Step 1: Solve Turnstile
	token, err := s.capSolver.SolveTurnstile(ctx, caBaseURL+"/IndividualNameSearch", caTurnstileSiteKey)
	if err != nil {
		return nil, fmt.Errorf("ca: captcha solve failed: %w", err)
	}

	// Step 2: GET search page to get cookies and CSRF token
	searchURL := caBaseURL + "/IndividualNameSearch"
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ca: get request error: %w", err)
	}
	getReq.Header = http.Header{
		"Accept":            {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		"User-Agent":        {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {"Accept", "Accept-Language", "User-Agent"},
	}

	resp, err := session.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("ca: get page failed: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ca: parse page failed: %w", err)
	}

	csrfToken, _ := doc.Find("input[name='__RequestVerificationToken']").Attr("value")

	// Step 3: POST search
	formData := url.Values{
		"SearchLastName":            {lastName},
		"SearchFirstName":           {firstName},
		"cf-turnstile-response":     {token},
		"__RequestVerificationToken": {csrfToken},
	}

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("ca: post request error: %w", err)
	}
	postReq.Header = http.Header{
		"Content-Type":      {"application/x-www-form-urlencoded"},
		"Referer":           {searchURL},
		"Accept":            {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		"User-Agent":        {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {"Content-Type", "Referer", "Accept", "Accept-Language", "User-Agent"},
	}

	postResp, err := session.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("ca: post search failed: %w", err)
	}
	defer postResp.Body.Close()

	resultDoc, err := goquery.NewDocumentFromReader(postResp.Body)
	if err != nil {
		return nil, fmt.Errorf("ca: parse results failed: %w", err)
	}

	// Step 4: Parse results table
	var results []LicenseResult
	resultDoc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		if i >= 5 {
			return
		}
		cells := row.Find("td")
		if cells.Length() < 3 {
			return
		}
		name := strings.TrimSpace(cells.Eq(0).Text())
		licNum := strings.TrimSpace(cells.Eq(1).Text())
		licType := strings.TrimSpace(cells.Eq(2).Text())
		status := ""
		if cells.Length() > 3 {
			status = strings.TrimSpace(cells.Eq(3).Text())
		}

		active := strings.Contains(strings.ToLower(status), "active")

		results = append(results, LicenseResult{
			Found:         true,
			Active:        active,
			FullName:      name,
			LicenseNumber: licNum,
			State:         "CA",
			LicenseType:   licType,
			Status:        status,
		})
	})

	if len(results) == 0 {
		return []LicenseResult{{Found: false, State: "CA"}}, nil
	}
	return results, nil
}

// LookupByNPN -- CA DOI doesn't support NPN search, return manual URL.
func (s *CaliforniaScraper) LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error) {
	return []LicenseResult{{
		Found: false,
		State: "CA",
		Error: "CA DOI does not support NPN search. Verify manually at " + s.ManualLookupURL(),
	}}, nil
}

// LookupByLicenseNumber -- CA DOI doesn't support license# search via this path, return manual URL.
func (s *CaliforniaScraper) LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error) {
	return []LicenseResult{{
		Found: false,
		State: "CA",
		Error: "CA DOI license number search not implemented. Verify manually at " + s.ManualLookupURL(),
	}}, nil
}
