package api

import (
	"net/http"
	"time"
)

// parseDateRange extracts from/to query params (YYYY-MM-DD) with a default of the current week.
func parseDateRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now()
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Default: current week (Monday–Sunday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	from := now.AddDate(0, 0, -(weekday - 1))
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 0, 6)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, now.Location())

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, now.Location())
		}
	}

	return from, to
}
