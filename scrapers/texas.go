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

const txBaseURL = "https://txapps.texas.gov/NASApp/tdi/TdiARManager"

type TexasScraper struct {
	sessionFactory SessionFactory
	capSolver      *captcha.CapSolver
}

func NewTexasScraper(sessionFactory SessionFactory, cs *captcha.CapSolver) *TexasScraper {
	return &TexasScraper{sessionFactory: sessionFactory, capSolver: cs}
}

func (s *TexasScraper) StateCode() string      { return "TX" }
func (s *TexasScraper) ManualLookupURL() string { return txBaseURL }

func (s *TexasScraper) LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error) {
	if s.capSolver == nil {
		return []LicenseResult{{
			Found: false,
			State: "TX",
			Error: "TX DOI requires CAPTCHA solving. CAPSOLVER_API_KEY not configured. Verify manually at " + s.ManualLookupURL(),
		}}, nil
	}

	log.Printf("TX lookup: %s %s", firstName, lastName)

	session, err := s.sessionFactory()
	if err != nil {
		return nil, fmt.Errorf("tx: session error: %w", err)
	}

	// Step 1: GET search page
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, txBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tx: get request error: %w", err)
	}
	getReq.Header = http.Header{
		"Accept":            {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		"User-Agent":        {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {"Accept", "Accept-Language", "User-Agent"},
	}

	resp, err := session.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("tx: get page failed: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tx: parse page failed: %w", err)
	}

	// Extract captchaToken
	captchaToken, _ := doc.Find("input[name='captchaToken']").Attr("value")

	// Step 2: Solve captcha (using Turnstile solver as fallback -- TX may use reCAPTCHA)
	// For TX, we try solving whatever captcha is present
	var solvedToken string
	if captchaToken != "" {
		// TX uses a captcha that CapSolver can handle
		solvedToken, err = s.capSolver.SolveTurnstile(ctx, txBaseURL, captchaToken)
		if err != nil {
			return nil, fmt.Errorf("tx: captcha solve failed: %w", err)
		}
	}

	// Step 3: POST search
	formData := url.Values{
		"perlastName":  {lastName},
		"perfirstName": {firstName},
		"captchaToken": {solvedToken},
		"search":       {"Search"},
	}

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, txBaseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("tx: post request error: %w", err)
	}
	postReq.Header = http.Header{
		"Content-Type":      {"application/x-www-form-urlencoded"},
		"Referer":           {txBaseURL},
		"Accept":            {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		"User-Agent":        {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {"Content-Type", "Referer", "Accept", "Accept-Language", "User-Agent"},
	}

	postResp, err := session.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("tx: post search failed: %w", err)
	}
	defer postResp.Body.Close()

	resultDoc, err := goquery.NewDocumentFromReader(postResp.Body)
	if err != nil {
		return nil, fmt.Errorf("tx: parse results failed: %w", err)
	}

	// Step 4: Parse results
	var results []LicenseResult
	resultDoc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		if i >= 5 {
			return
		}
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}
		name := strings.TrimSpace(cells.Eq(0).Text())
		licNum := strings.TrimSpace(cells.Eq(1).Text())
		status := ""
		if cells.Length() > 2 {
			status = strings.TrimSpace(cells.Eq(2).Text())
		}
		licType := ""
		if cells.Length() > 3 {
			licType = strings.TrimSpace(cells.Eq(3).Text())
		}

		active := strings.Contains(strings.ToLower(status), "active")

		results = append(results, LicenseResult{
			Found:         true,
			Active:        active,
			FullName:      name,
			LicenseNumber: licNum,
			State:         "TX",
			LicenseType:   licType,
			Status:        status,
		})
	})

	if len(results) == 0 {
		return []LicenseResult{{Found: false, State: "TX"}}, nil
	}
	return results, nil
}

func (s *TexasScraper) LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error) {
	return []LicenseResult{{
		Found: false,
		State: "TX",
		Error: "TX DOI does not support NPN search. Verify manually at " + s.ManualLookupURL(),
	}}, nil
}

func (s *TexasScraper) LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error) {
	return []LicenseResult{{
		Found: false,
		State: "TX",
		Error: "TX DOI license number search not implemented. Verify manually at " + s.ManualLookupURL(),
	}}, nil
}
