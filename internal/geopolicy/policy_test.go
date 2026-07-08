package geopolicy

import (
	"testing"

	"anti-fraud/internal/geoiputil"
)

func TestEvaluateMatchesBlockedCountry(t *testing.T) {
	cfg := Config{
		BlockedCountries: map[string]struct{}{
			"NL": {},
		},
	}

	match := cfg.Evaluate(geoiputil.LookupResult{CountryISO: "nl"})
	if !match.Blocked {
		t.Fatal("expected country-based GeoIP policy match")
	}
	if match.MatchedCountry != "NL" {
		t.Fatalf("expected matched country NL, got %q", match.MatchedCountry)
	}
}

func TestEvaluateMatchesBlockedASNNumber(t *testing.T) {
	cfg := Config{
		BlockedASNNumbers: map[uint]struct{}{
			15169: {},
		},
	}

	match := cfg.Evaluate(geoiputil.LookupResult{ASNNumber: 15169})
	if !match.Blocked {
		t.Fatal("expected ASN number-based GeoIP policy match")
	}
	if match.MatchedASNNumber != 15169 {
		t.Fatalf("expected matched ASN 15169, got %d", match.MatchedASNNumber)
	}
}

func TestEvaluateMatchesBlockedASNKeyword(t *testing.T) {
	cfg := Config{
		BlockedASNKeywords: []string{"digitalocean", "cloudflare"},
	}

	match := cfg.Evaluate(geoiputil.LookupResult{ASNOrg: "DigitalOcean, LLC"})
	if !match.Blocked {
		t.Fatal("expected ASN keyword-based GeoIP policy match")
	}
	if match.MatchedASNKeyword != "digitalocean" {
		t.Fatalf("expected matched keyword digitalocean, got %q", match.MatchedASNKeyword)
	}
}

func TestEvaluateAllowsNonMatchingLookup(t *testing.T) {
	cfg := Config{
		BlockedCountries: map[string]struct{}{
			"NL": {},
		},
		BlockedASNNumbers: map[uint]struct{}{
			15169: {},
		},
		BlockedASNKeywords: []string{"cloudflare"},
	}

	match := cfg.Evaluate(geoiputil.LookupResult{
		CountryISO: "US",
		ASNNumber:  12345,
		ASNOrg:     "Example ISP",
	})
	if match.Blocked {
		t.Fatalf("expected lookup to pass, got %+v", match)
	}
}
