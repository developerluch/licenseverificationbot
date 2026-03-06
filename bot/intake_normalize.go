package bot

import (
	"strings"
)

// normalizeAgency normalizes agency names to canonical forms.
func normalizeAgency(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lower == "tfc" || lower == "topfloorclosers" || lower == "top floor closers":
		return "TFC"
	case lower == "radiant" || lower == "radiant financial":
		return "Radiant"
	case lower == "gbu":
		return "GBU"
	case strings.Contains(lower, "trulight") || strings.Contains(lower, "tru light"):
		return "TruLight"
	case strings.Contains(lower, "thrive"):
		return "Thrive"
	case strings.Contains(lower, "the point") || lower == "thepoint":
		return "The Point"
	case strings.Contains(lower, "synergy"):
		return "Synergy"
	case strings.Contains(lower, "illuminate"):
		return "Illuminate"
	case strings.Contains(lower, "elite one") || strings.Contains(lower, "elite 1") || lower == "eliteone":
		return "Elite One"
	default:
		return "Other"
	}
}

// normalizeLicenseStatus normalizes license status values.
func normalizeLicenseStatus(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "licensed") || lower == "yes":
		return "licensed"
	case strings.Contains(lower, "study") || strings.Contains(lower, "studying"):
		return "studying"
	default:
		return "none"
	}
}

// normalizeExperience normalizes experience level values.
func normalizeExperience(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "2+") || strings.Contains(lower, "2 year") || strings.Contains(lower, "2yr"):
		return "2yr_plus"
	case strings.Contains(lower, "1-2") || strings.Contains(lower, "1 to 2") || strings.Contains(lower, "1yr"):
		return "1_2yr"
	case strings.Contains(lower, "6-12") || strings.Contains(lower, "6 to 12"):
		return "6_12mo"
	case strings.Contains(lower, "<6") || strings.Contains(lower, "less than 6") || strings.Contains(lower, "6 mo"):
		return "less_6mo"
	default:
		return "none"
	}
}

// normalizeLeadSource normalizes lead source values.
func normalizeLeadSource(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "both"):
		return "both"
	case strings.Contains(lower, "buy") || strings.Contains(lower, "own"):
		return "buy_own"
	case strings.Contains(lower, "agency") || strings.Contains(lower, "funded"):
		return "agency_funded"
	default:
		if raw == "" {
			return ""
		}
		return raw
	}
}

// normalizeShowComp normalizes the "show comp" boolean value.
func normalizeShowComp(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return lower == "yes" || lower == "true" || lower == "1"
}
