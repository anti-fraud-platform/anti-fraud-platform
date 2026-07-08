package headercheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScore_RealisticChromeRequest_NotSuspicious(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/click", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="125", "Google Chrome";v="125"`)

	res := Score(req)
	if res.IsSuspicious() {
		t.Fatalf("expected realistic browser request to pass, got score %d, reasons %v", res.Score, res.Reasons)
	}
}

func TestScore_BrowserFetchDefaultAcceptWildcard_NotFlaggedByAcceptAlone(t *testing.T) {
	// This is the exact false-positive an earlier version of this file had:
	// fetch() with no explicit Accept header sends "Accept: */*" by browser
	// default. A same-origin fetch() call from the clicker page (full
	// Sec-Fetch-*, Accept-Language, Accept-Encoding, Sec-Ch-Ua present)
	// must NOT be flagged just because Accept is "*/*".
	req := httptest.NewRequest(http.MethodPost, "/v1/click", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="125", "Google Chrome";v="125"`)

	res := Score(req)
	if res.IsSuspicious() {
		t.Fatalf("expected wildcard Accept with all other browser signals present to pass, got score %d, reasons %v", res.Score, res.Reasons)
	}
}

func TestScore_PlainGoHTTPClient_Suspicious(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/click", nil)
	req.Header.Set("User-Agent", "Go-http-client/1.1")

	res := Score(req)
	if !res.IsSuspicious() {
		t.Fatalf("expected bare HTTP client request to be flagged, got score %d", res.Score)
	}
}

func TestScore_SpoofedChromeUAWithoutClientHints_Suspicious(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/click", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	res := Score(req)
	if !res.IsSuspicious() {
		t.Fatalf("expected spoofed Chrome UA without client hints to be flagged, got score %d, reasons %v", res.Score, res.Reasons)
	}
}

func TestScore_CurlUA_Suspicious(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/click", nil)
	req.Header.Set("User-Agent", "curl/8.4.0")

	res := Score(req)
	if !res.IsSuspicious() {
		t.Fatalf("expected curl request to be flagged, got score %d", res.Score)
	}
}
