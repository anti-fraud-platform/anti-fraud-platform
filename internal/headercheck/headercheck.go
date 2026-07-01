// Package headercheck scores an HTTP request on how closely its header set
// resembles a real browser vs. a bare HTTP client library (curl,
// python-requests, axios, Go's net/http with no customization).
//
// This is a SCORE, not a single boolean check, because no one missing
// header proves anything on its own. Accumulating weak evidence across
// several signals is the standard approach real anti-fraud heuristics use.
//
// Correctness note: an earlier draft of this file also flagged
// `Accept: */*` as suspicious. That is WRONG for this deployment — the
// clicker page's own fetch() calls send exactly that Accept value by
// browser default, which would have flagged legitimate traffic. Removed;
// Accept-Language, Accept-Encoding, and Sec-Fetch-* carry the signal
// instead, since curl/python-requests/axios send none of those by default.
package headercheck

import (
	"net/http"
	"strings"
)

// Result is the outcome of scoring a single request.
type Result struct {
	Score   int
	Reasons []string
}

// Threshold is the score at or above which a request is flagged as
// suspicious. Starting point — tune against your own traffic before
// relying on it for reported numbers.
const Threshold = 4

// Score inspects headers a real browser sends by default but that scripted
// HTTP clients usually omit or send inconsistently.
func Score(r *http.Request) Result {
	res := Result{}

	if r.Header.Get("Accept") == "" {
		res.Score++
		res.Reasons = append(res.Reasons, "missing_accept")
	}
	if r.Header.Get("Accept-Language") == "" {
		res.Score++
		res.Reasons = append(res.Reasons, "missing_accept_language")
	}
	if r.Header.Get("Accept-Encoding") == "" {
		res.Score++
		res.Reasons = append(res.Reasons, "missing_accept_encoding")
	}
	if r.Header.Get("Sec-Fetch-Site") == "" {
		res.Score++
		res.Reasons = append(res.Reasons, "missing_sec_fetch_site")
	}
	if r.Header.Get("Sec-Fetch-Mode") == "" {
		res.Score++
		res.Reasons = append(res.Reasons, "missing_sec_fetch_mode")
	}

	ua := strings.ToLower(r.Header.Get("User-Agent"))
	claimsChromium := strings.Contains(ua, "chrome") || strings.Contains(ua, "chromium")
	if claimsChromium && r.Header.Get("Sec-Ch-Ua") == "" {
		// Real Chromium browsers have sent Client Hints (Sec-Ch-Ua) by
		// default since Chrome 89 (2021), on every request including
		// same-origin fetch — not just navigations. A UA string claiming
		// Chrome with zero client hints is a strong sign the UA header was
		// set manually by a script trying to dodge UA sniffing.
		res.Score += 2
		res.Reasons = append(res.Reasons, "chrome_ua_without_client_hints")
	}

	return res
}

// IsSuspicious reports whether the score meets or exceeds Threshold.
func (r Result) IsSuspicious() bool {
	return r.Score >= Threshold
}
