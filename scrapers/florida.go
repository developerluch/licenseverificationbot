package scrapers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/PuerkitoBio/goquery"
)

const flBaseURL = "https://licenseesearch.fldfs.com"

// FloridaScraper scrapes the Florida Department of Financial Services license search.
type FloridaScraper struct {
	sessionFactory SessionFactory
}

// NewFloridaScraper creates a new Florida DOI scraper.
func NewFloridaScraper(sessionFactory SessionFactory) *FloridaScraper {
	return &FloridaScraper{sessionFactory: sessionFactory}
}

// StateCode returns "FL".
func (s *FloridaScraper) StateCode() string { return "FL" }

// ManualLookupURL returns the FL DOI search URL.
func (s *FloridaScraper) ManualLookupURL() string { return flBaseURL }

// LookupByName searches FL DOI by first + last name.
func (s *FloridaScraper) LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error) {
	log.Printf("FL lookup: %s %s", firstName, lastName)
	form := s.buildFormData(firstName, lastName, "", "")
	return s.search(ctx, form)
}

// LookupByNPN searches FL DOI by National Producer Number.
func (s *FloridaScraper) LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error) {
	log.Printf("FL NPN lookup: %s", npn)
	form := s.buildFormData("", "", "", npn)
	return s.search(ctx, form)
}

// LookupByLicenseNumber searches FL DOI by FL license number.
func (s *FloridaScraper) LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error) {
	log.Printf("FL license# lookup: %s", licenseNumber)
	form := s.buildFormData("", "", licenseNumber, "")
	return s.search(ctx, form)
}

// buildFormData builds the full POST payload matching the FL DOI form exactly.
func (s *FloridaScraper) buildFormData(firstName, lastName, licenseNumber, npn string) url.Values {
	return url.Values{
		"IndividualFNameFilter":                          {strings.TrimSpace(firstName)},
		"IndividualLNameFilter":                          {strings.TrimSpace(lastName)},
		"IndividualMNameFilter":                          {""},
		"EmailAddressBeginContainFilter":                 {"1"},
		"EmailFilter":                                    {""},
		"FirmNameBeginContainFilter":                     {"1"},
		"FirmNameFilter":                                 {""},
		"ResidentStatusFilter":                           {""},
		"FLLicenseNoFilter":                              {strings.TrimSpace(licenseNumber)},
		"NPNNoFilter":                                    {strings.TrimSpace(npn)},
		"LicenseStatusFilter":                            {"1"},
		"LicenseCategoryFilter":                          {""},
		"LicenseIssueDateFromFilter":                     {""},
		"LicenseIssueDateToFilter":                       {""},
		"OnlyLicWithNoQuApptFilter":                      {"false"},
		"BusinessStateFilter":                            {""},
		"BusinessCityFilter":                             {""},
		"BusinessCountyFilter":                           {""},
		"BusinessZipFilter":                              {""},
		"CEDueDtFromFilter":                              {""},
		"CEDueDtToFilter":                                {""},
		"CEHrsNotMetFilter":                              {"false"},
		"AppointingEntityTYCLFilter":                     {""},
		"AppointingEntityStatusFilter":                   {""},
		"AppointingEntityStatusDateFromFilter":            {""},
		"AppointingEntityStatusDateToFilter":              {""},
		"LicenseeSearchInfo.PagingInfo.SortBy":           {"Name"},
		"LicenseeSearchInfo.PagingInfo.SortDesc":         {"False"},
		"LicenseeSearchInfo.PagingInfo.CurrentPage":      {"1"},
		"AppointingEntityIdFilter":                       {""},
		"AppointingEntityDisplayName":                    {""},
		"TabLLValue":                                     {"0"},
		"TabCEValue":                                     {"0"},
		"TabAppValue":                                    {""},
		"hdnLApptEntitySearchListUrl":                    {"/Home/GetAppointingEntityListForSearch"},
		"hdnLicenseeSearchListUrl":                       {"/Home/GetLicenseeSearchListPartialView"},
	}
}

// search executes the two-step search: GET to establish cookies, then POST form to /.
func (s *FloridaScraper) search(ctx context.Context, form url.Values) ([]LicenseResult, error) {
	session, err := s.sessionFactory()
	if err != nil {
		return nil, fmt.Errorf("fl: session error: %w", err)
	}

	// Step 1: GET / to establish session cookies
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, flBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fl: GET request build error: %w", err)
	}
	getReq.Header = http.Header{
		"Accept":            {"text/html"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		http.HeaderOrderKey: {"Accept", "Accept-Language"},
	}

	getResp, err := session.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("fl: GET request failed: %w", err)
	}
	io.Copy(io.Discard, getResp.Body)
	getResp.Body.Close()

	if getResp.StatusCode != 200 {
		return []LicenseResult{{Error: fmt.Sprintf("HTTP %d on GET", getResp.StatusCode)}}, nil
	}

	// Step 2: POST / with full form payload
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, flBaseURL+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("fl: POST request build error: %w", err)
	}
	postReq.Header = http.Header{
		"Content-Type":      {"application/x-www-form-urlencoded"},
		"Referer":           {flBaseURL},
		"Accept":            {"text/html"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		http.HeaderOrderKey: {"Content-Type", "Referer", "Accept", "Accept-Language"},
	}

	postResp, err := session.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("fl: POST request failed: %w", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != 200 {
		return []LicenseResult{{Error: fmt.Sprintf("HTTP %d on POST", postResp.StatusCode)}}, nil
	}

	// Read the full body so we can check for "no results" text and also parse HTML
	bodyBytes, err := io.ReadAll(postResp.Body)
	if err != nil {
		return nil, fmt.Errorf("fl: read POST body failed: %w", err)
	}
	html := string(bodyBytes)

	return s.parseSearchResults(ctx, session, html)
}

// parseSearchResults parses the FL DOI search results page and fetches detail pages.
func (s *FloridaScraper) parseSearchResults(ctx context.Context, session interface{ Do(req *http.Request) (*http.Response, error) }, html string) ([]LicenseResult, error) {
	lowerHTML := strings.ToLower(html)
	if strings.Contains(lowerHTML, "no licensee") || strings.Contains(lowerHTML, "no results") {
		log.Println("FL: No results found")
		return []LicenseResult{{Found: false, State: "FL"}}, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("fl: parse HTML error: %w", err)
	}

	table := doc.Find("table.table")
	if table.Length() == 0 {
		log.Println("FL: Could not find results table")
		return []LicenseResult{{Found: false, State: "FL"}}, nil
	}

	tbody := table.First().Find("tbody")
	if tbody.Length() == 0 {
		return []LicenseResult{{Found: false, State: "FL"}}, nil
	}

	var results []LicenseResult
	tbody.Find("tr").Each(func(i int, row *goquery.Selection) {
		if len(results) >= 5 {
			return
		}

		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameCell := cells.Eq(0)
		licenseCell := cells.Eq(1)

		link := nameCell.Find("a")
		if link.Length() == 0 {
			return
		}

		fullName := strings.TrimSpace(link.Text())
		detailPath, exists := link.Attr("href")
		if !exists {
			return
		}
		licenseNumber := strings.TrimSpace(licenseCell.Text())

		// Fetch the detail page for status, type, expiration
		detail := s.fetchDetail(ctx, session, detailPath)

		status := detail["status"]
		result := LicenseResult{
			Found:           true,
			State:           "FL",
			FullName:        fullName,
			LicenseNumber:   licenseNumber,
			Status:          status,
			Active:          strings.EqualFold(status, "VALID"),
			LicenseType:     detail["type"],
			NPN:             detail["npn"],
			ExpirationDate:  detail["expiration"],
			IssueDate:       detail["issue_date"],
			BusinessAddress: detail["business_address"],
			BusinessPhone:   detail["phone"],
			Email:           detail["email"],
			County:          detail["county"],
		}
		results = append(results, result)
	})

	if len(results) == 0 {
		return []LicenseResult{{Found: false, State: "FL"}}, nil
	}

	log.Printf("FL: Found %d results", len(results))
	return results, nil
}

// fetchDetail fetches a licensee detail page and extracts status, type, NPN, etc.
//
// The FL DOI detail page layout:
//   - Top section: form-group divs with label + sibling div for values
//     (License #, Full Name, Business Address, Email, Phone, County, NPN #)
//   - "Valid Licenses" panel: table with Type | Issue Date | Qualifying Appointment
//   - "Invalid Licenses" panel: same table structure
//   - "Active Appointments" / "Inactive Appointments" panels
//
// Status is determined by WHICH panel the license appears in (Valid vs Invalid).
func (s *FloridaScraper) fetchDetail(ctx context.Context, session interface{ Do(req *http.Request) (*http.Response, error) }, detailPath string) map[string]string {
	result := map[string]string{}

	detailURL := flBaseURL + detailPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		log.Printf("FL detail request build error: %v", err)
		result["error"] = err.Error()
		return result
	}
	req.Header = http.Header{
		"Accept":            {"text/html"},
		"Referer":           {flBaseURL + "/"},
		"Accept-Language":   {"en-US,en;q=0.9"},
		http.HeaderOrderKey: {"Accept", "Referer", "Accept-Language"},
	}

	resp, err := session.Do(req)
	if err != nil {
		log.Printf("FL detail request failed: %v", err)
		result["error"] = err.Error()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		result["error"] = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("FL detail parse error: %v", err)
		result["error"] = err.Error()
		return result
	}

	// ── Extract label:value pairs from form-group divs ──
	// Structure:
	//   <div class="form-group">
	//     <label class="control-label">NPN #:</label>
	//     <div class="col-md-8">12345</div>
	//   </div>
	formFields := map[string]string{}
	doc.Find("div.form-group").Each(func(_ int, fg *goquery.Selection) {
		labelTag := fg.Find("label")
		if labelTag.Length() == 0 {
			return
		}
		labelText := strings.TrimSpace(labelTag.Text())
		labelText = strings.TrimRight(labelText, ":")

		// Value div is the sibling of the label inside the same form-group
		valueDiv := labelTag.Next()
		if valueDiv.Length() == 0 {
			return
		}
		value := strings.TrimSpace(valueDiv.Text())
		if value == "" {
			return
		}

		key := strings.ToLower(labelText)
		key = strings.ReplaceAll(key, " ", "_")
		key = strings.ReplaceAll(key, "#", "num")
		formFields[key] = value
	})

	result["full_name"] = formFields["full_name"]
	if v, ok := formFields["license_num"]; ok {
		result["license_number"] = v
	} else {
		result["license_number"] = formFields["license"]
	}
	result["npn"] = formFields["npn_num"]
	result["email"] = formFields["email"]
	result["phone"] = formFields["phone"]
	result["business_address"] = formFields["business_address"]
	result["mailing_address"] = formFields["mailing_address"]
	result["county"] = formFields["county"]

	// ── Extract license types from Valid / Invalid panels ──
	type licenseEntry struct {
		licType   string
		issueDate string
		status    string
	}
	var validLicenses []licenseEntry
	var invalidLicenses []licenseEntry

	doc.Find("div.panel").Each(func(_ int, panel *goquery.Selection) {
		heading := panel.Find("div.panel-heading")
		if heading.Length() == 0 {
			return
		}
		headingText := strings.ToLower(strings.TrimSpace(heading.Text()))

		table := panel.Find("table")
		if table.Length() == 0 {
			return
		}
		tbody := table.Find("tbody")
		if tbody.Length() == 0 {
			return
		}

		tbody.Find("tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}
			licType := strings.TrimSpace(cells.Eq(0).Text())
			issueDate := strings.TrimSpace(cells.Eq(1).Text())
			entry := licenseEntry{licType: licType, issueDate: issueDate}

			if strings.Contains(headingText, "valid license") {
				entry.status = "VALID"
				validLicenses = append(validLicenses, entry)
			} else if strings.Contains(headingText, "invalid license") {
				entry.status = "INVALID"
				invalidLicenses = append(invalidLicenses, entry)
			}
		})
	})

	// Pick the best license to report:
	// Prefer life/health insurance from valid licenses, then any valid, then invalid
	var chosen *licenseEntry
	for i := range validLicenses {
		lower := strings.ToLower(validLicenses[i].licType)
		if strings.Contains(lower, "life") || strings.Contains(lower, "health") {
			chosen = &validLicenses[i]
			break
		}
	}
	if chosen == nil && len(validLicenses) > 0 {
		chosen = &validLicenses[0]
	}
	if chosen == nil && len(invalidLicenses) > 0 {
		chosen = &invalidLicenses[0]
	}

	if chosen != nil {
		result["type"] = chosen.licType
		result["issue_date"] = chosen.issueDate
		result["status"] = chosen.status
	}

	// ── Extract expiration from Active/Inactive Appointments ──
	doc.Find("div.panel").Each(func(_ int, panel *goquery.Selection) {
		heading := panel.Find("div.panel-heading")
		if heading.Length() == 0 {
			return
		}
		headingText := strings.ToLower(strings.TrimSpace(heading.Text()))
		if !strings.Contains(headingText, "appointment") {
			return
		}

		table := panel.Find("table")
		if table.Length() == 0 {
			return
		}
		tbody := table.Find("tbody")
		if tbody.Length() == 0 {
			return
		}

		tbody.Find("tr").Each(func(_ int, row *goquery.Selection) {
			if result["expiration"] != "" {
				return
			}
			cells := row.Find("td")
			// Active Appointments: Company Name | Issue Date | Exp Date | Status Date
			if cells.Length() >= 3 {
				expDate := strings.TrimSpace(cells.Eq(2).Text())
				if expDate != "" {
					result["expiration"] = expDate
				}
			}
		})
	})

	return result
}
