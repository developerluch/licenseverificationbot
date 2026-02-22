//go:build integration

package scrapers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"license-bot-go/tlsclient"
)

// TestNAICCoverage tests which manual-lookup states are actually available
// through the NAIC SBS API. Run with: go test -tags=integration -run TestNAICCoverage -v ./scrapers/
//
// For each state in ManualLookupURLs, it queries the NAIC API with a common
// last name ("Smith") and checks if results come back. States that return
// results can be moved from ManualLookupURLs to NAICStates.
func TestNAICCoverage(t *testing.T) {
	client := tlsclient.New()

	// Test all states currently in ManualLookupURLs
	manualStates := []string{
		"CO", "GA", "IN", "KY", "LA", "ME", "MI", "MN",
		"MS", "NV", "NY", "OH", "PA", "UT", "VA", "WA", "WY",
		// Also test territories
		"PR", "GU", "VI", "MP", "AS",
	}

	var working []string
	var notWorking []string

	for _, state := range manualStates {
		t.Run(state, func(t *testing.T) {
			scraper := NewNAICScraper(client.NewSession, state)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := scraper.LookupByName(ctx, "John", "Smith")
			if err != nil {
				t.Logf("%s: ERROR -- %v", state, err)
				notWorking = append(notWorking, state)
				return
			}

			foundCount := 0
			for _, r := range results {
				if r.Found {
					foundCount++
				}
			}

			if foundCount > 0 {
				t.Logf("%s: WORKS -- %d results found", state, foundCount)
				working = append(working, state)
			} else {
				t.Logf("%s: NO RESULTS (API responded but no matches)", state)
				notWorking = append(notWorking, state)
			}

			// Rate limit: be polite to the NAIC API
			time.Sleep(2 * time.Second)
		})
	}

	fmt.Println("\n=== NAIC Coverage Report ===")
	fmt.Printf("Working states (move to NAICStates): %v\n", working)
	fmt.Printf("Not working (keep manual): %v\n", notWorking)
}

// TestNAICExistingStates verifies that all currently configured NAIC states still work.
func TestNAICExistingStates(t *testing.T) {
	client := tlsclient.New()

	// Sample a few from the existing 31 states
	sampleStates := []string{"AL", "IL", "MA", "NC", "TN"}

	for _, state := range sampleStates {
		t.Run(state, func(t *testing.T) {
			scraper := NewNAICScraper(client.NewSession, state)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := scraper.LookupByName(ctx, "John", "Smith")
			if err != nil {
				t.Errorf("%s: ERROR -- %v", state, err)
				return
			}

			foundCount := 0
			for _, r := range results {
				if r.Found {
					foundCount++
				}
			}

			if foundCount == 0 {
				t.Errorf("%s: expected results for common name but got none", state)
			} else {
				t.Logf("%s: OK -- %d results", state, foundCount)
			}

			time.Sleep(2 * time.Second)
		})
	}
}
