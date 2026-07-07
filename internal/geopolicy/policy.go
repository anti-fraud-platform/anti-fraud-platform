package geopolicy

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"anti-fraud/internal/geoiputil"
)

type Config struct {
	BlockedCountries   map[string]struct{}
	BlockedASNNumbers  map[uint]struct{}
	BlockedASNKeywords []string
}

type Match struct {
	Blocked           bool
	Reason            string
	MatchedCountry    string
	MatchedASNNumber  uint
	MatchedASNKeyword string
}

func FromEnv() (Config, error) {
	cfg := Config{
		BlockedCountries:  map[string]struct{}{},
		BlockedASNNumbers: map[uint]struct{}{},
	}

	for _, token := range splitCSV(os.Getenv("GEOIP_BLOCKED_COUNTRIES")) {
		cfg.BlockedCountries[strings.ToUpper(token)] = struct{}{}
	}

	for _, token := range splitCSV(os.Getenv("GEOIP_BLOCKED_ASN_NUMBERS")) {
		asn, err := strconv.ParseUint(token, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid GEOIP_BLOCKED_ASN_NUMBERS value %q: %w", token, err)
		}
		cfg.BlockedASNNumbers[uint(asn)] = struct{}{}
	}

	for _, token := range splitCSV(os.Getenv("GEOIP_BLOCKED_ASN_KEYWORDS")) {
		cfg.BlockedASNKeywords = append(cfg.BlockedASNKeywords, strings.ToLower(token))
	}

	return cfg, nil
}

func (c Config) Enabled() bool {
	return len(c.BlockedCountries) > 0 || len(c.BlockedASNNumbers) > 0 || len(c.BlockedASNKeywords) > 0
}

func (c Config) Summary() string {
	countries := make([]string, 0, len(c.BlockedCountries))
	for country := range c.BlockedCountries {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	asnNumbers := make([]int, 0, len(c.BlockedASNNumbers))
	for asn := range c.BlockedASNNumbers {
		asnNumbers = append(asnNumbers, int(asn))
	}
	sort.Ints(asnNumbers)

	asnStrings := make([]string, 0, len(asnNumbers))
	for _, asn := range asnNumbers {
		asnStrings = append(asnStrings, strconv.Itoa(asn))
	}

	return fmt.Sprintf(
		"countries=%s asn_numbers=%s asn_keywords=%s",
		renderList(countries),
		renderList(asnStrings),
		renderList(c.BlockedASNKeywords),
	)
}

func (c Config) Evaluate(lookup geoiputil.LookupResult) Match {
	country := strings.ToUpper(strings.TrimSpace(lookup.CountryISO))
	if country != "" {
		if _, blocked := c.BlockedCountries[country]; blocked {
			return Match{
				Blocked:        true,
				Reason:         "geoip_policy",
				MatchedCountry: country,
			}
		}
	}

	if lookup.ASNNumber != 0 {
		if _, blocked := c.BlockedASNNumbers[lookup.ASNNumber]; blocked {
			return Match{
				Blocked:          true,
				Reason:           "geoip_policy",
				MatchedASNNumber: lookup.ASNNumber,
			}
		}
	}

	asnOrg := strings.ToLower(strings.TrimSpace(lookup.ASNOrg))
	if asnOrg != "" {
		for _, keyword := range c.BlockedASNKeywords {
			if strings.Contains(asnOrg, keyword) {
				return Match{
					Blocked:           true,
					Reason:            "geoip_policy",
					MatchedASNKeyword: keyword,
				}
			}
		}
	}

	return Match{}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

func renderList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	return "[" + strings.Join(items, ", ") + "]"
}
