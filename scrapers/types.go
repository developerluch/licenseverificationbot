package scrapers

import (
	"context"
	"strings"
)

// LicenseResult holds the standardized result from any state DOI lookup.
type LicenseResult struct {
	Found           bool
	Active          bool
	Resident        bool
	FullName        string
	LicenseNumber   string
	NPN             string
	State           string
	LicenseType     string
	Status          string
	ExpirationDate  string
	IssueDate       string
	LOAs            string
	BusinessAddress string
	BusinessPhone   string
	Email           string
	County          string
	Error           string
}

// IsLifeLicensed returns true if the license is active and covers life insurance.
// Note: "life" substring matching is sufficient for insurance license types.
// False positives from words like "nightlife" don't occur in DOI/NAIC license fields.
func (r LicenseResult) IsLifeLicensed() bool {
	if !r.Active {
		return false
	}
	lower := strings.ToLower(r.LicenseType) + " " + strings.ToLower(r.LOAs)
	return strings.Contains(lower, "life")
}

// Scraper is the interface every state scraper must implement.
type Scraper interface {
	StateCode() string
	LookupByName(ctx context.Context, firstName, lastName string) ([]LicenseResult, error)
	LookupByNPN(ctx context.Context, npn string) ([]LicenseResult, error)
	LookupByLicenseNumber(ctx context.Context, licenseNumber string) ([]LicenseResult, error)
	ManualLookupURL() string
}
