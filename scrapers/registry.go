package scrapers

import (
	"context"
	"fmt"
	"strings"

	"license-bot-go/scrapers/captcha"
	"license-bot-go/tlsclient"
)

// NAICStates -- the 31 states covered by the NAIC SBS API.
var NAICStates = map[string]bool{
	"AL": true, "AK": true, "AZ": true, "AR": true, "CT": true,
	"DE": true, "DC": true, "HI": true, "ID": true, "IL": true,
	"IA": true, "KS": true, "MA": true, "MD": true, "MO": true,
	"MT": true, "NE": true, "NH": true, "NJ": true, "NM": true,
	"NC": true, "ND": true, "OK": true, "OR": true, "RI": true,
	"SC": true, "SD": true, "TN": true, "VT": true, "WI": true,
	"WV": true,
}

// ManualLookupURLs -- states that require manual lookup (no API, no CAPTCHA solving).
var ManualLookupURLs = map[string]string{
	"CO": "https://sircon.com/ComplianceExpress/",
	"GA": "https://sircon.com/ComplianceExpress/",
	"IN": "https://sircon.com/ComplianceExpress/",
	"KY": "https://insurance.ky.gov/ppc/agentlookup.aspx",
	"LA": "https://www.ldi.la.gov/producers/agent-search",
	"ME": "https://www.maine.gov/pfr/insurance/licensee-search",
	"MI": "https://difs.state.mi.us/Licensees/",
	"MN": "https://sircon.com/ComplianceExpress/",
	"MS": "https://www.mid.ms.gov/licensing/agent-search.aspx",
	"NV": "https://doi.nv.gov/Licensing/Agent_Lookup/",
	"NY": "https://myportal.dfs.ny.gov/web/guest/individual-or-entity-look-up",
	"OH": "https://gateway.insurance.ohio.gov/UI/ODI.Agent.Public/",
	"PA": "https://sircon.com/ComplianceExpress/",
	"UT": "https://insurance.utah.gov/licensee-search/",
	"VA": "https://scc.virginia.gov/pages/Bureau-of-Insurance",
	"WA": "https://www.insurance.wa.gov/agent-broker-search",
	"WY": "https://sircon.com/ComplianceExpress/",
}

// Registry routes state codes to the appropriate scraper.
type Registry struct {
	tlsClient *tlsclient.Client
	capSolver *captcha.CapSolver
}

// NewRegistry creates a new scraper registry.
func NewRegistry(tlsClient *tlsclient.Client, capSolver *captcha.CapSolver) *Registry {
	return &Registry{
		tlsClient: tlsClient,
		capSolver: capSolver,
	}
}

// GetScraper returns the appropriate scraper for a state code.
func (r *Registry) GetScraper(stateCode string) Scraper {
	stateCode = strings.ToUpper(strings.TrimSpace(stateCode))

	switch stateCode {
	case "FL":
		return NewFloridaScraper(r.tlsClient.NewSession)
	case "CA":
		return NewCaliforniaScraper(r.tlsClient.NewSession, r.capSolver)
	case "TX":
		return NewTexasScraper(r.tlsClient.NewSession, r.capSolver)
	default:
		if NAICStates[stateCode] {
			return NewNAICScraper(r.tlsClient.NewSession, stateCode)
		}
		if url, ok := ManualLookupURLs[stateCode]; ok {
			return NewManualScraper(stateCode, url)
		}
		// Unknown state -- return manual scraper with empty URL
		return NewManualScraper(stateCode, "")
	}
}

// ManualScraper returns a result directing users to manually look up their license.
type ManualScraper struct {
	stateCode string
	lookupURL string
}

// NewManualScraper creates a new ManualScraper for the given state.
func NewManualScraper(stateCode, lookupURL string) *ManualScraper {
	return &ManualScraper{stateCode: stateCode, lookupURL: lookupURL}
}

// StateCode returns the two-letter state code.
func (s *ManualScraper) StateCode() string { return s.stateCode }

// ManualLookupURL returns the URL for manual license lookups.
func (s *ManualScraper) ManualLookupURL() string { return s.lookupURL }

// LookupByName returns a manual-lookup result for a name search.
func (s *ManualScraper) LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error) {
	return s.manualResult(), nil
}

// LookupByNPN returns a manual-lookup result for an NPN search.
func (s *ManualScraper) LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error) {
	return s.manualResult(), nil
}

// LookupByLicenseNumber returns a manual-lookup result for a license number search.
func (s *ManualScraper) LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error) {
	return s.manualResult(), nil
}

func (s *ManualScraper) manualResult() []LicenseResult {
	msg := fmt.Sprintf("Automated lookup not available for %s.", s.stateCode)
	if s.lookupURL != "" {
		msg += " Please verify manually at " + s.lookupURL
	}
	return []LicenseResult{{
		Found: false,
		State: s.stateCode,
		Error: msg,
	}}
}
