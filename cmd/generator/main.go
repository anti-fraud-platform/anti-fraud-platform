package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ClickEvent mirrors the payload expected by POST /v1/click
type ClickEvent struct {
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	CampaignID string `json:"campaign_id"`
	Timestamp  int64  `json:"timestamp"`
}

// --- random data pools ---

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
	"python-requests/2.31.0",
	"curl/8.6.0",
	"Go-http-client/1.1",
	"bot/2.0 (+http://example.com/bot)",
}

var campaignIDs = []string{
	"camp_alpha_001", "camp_beta_002", "camp_gamma_003",
	"camp_delta_004", "camp_epsilon_005", "camp_zeta_006",
}

func randomIP(rng *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rng.Intn(223)+1,
		rng.Intn(255),
		rng.Intn(255),
		rng.Intn(254)+1,
	)
}

// loadBlacklistIPs reads one IP per line from path. Blank lines and
// "#" comments are skipped. Used to inject real known-bad IPs into
// the simulated traffic stream so we can prove the Bloom filter
// drops them.
func loadBlacklistIPs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ips []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ips = append(ips, line)
	}
	return ips, scanner.Err()
}

// --- traffic profile ---
// Controls how each event is generated. This is what lets us move
// beyond simple "random everything" traffic into the more realistic
// and more dangerous patterns real fraud actually looks like.
type profile struct {
	blacklistIPs    []string // pool of real dirty IPs, loaded from file
	blacklistChance float64  // 0.0–1.0, probability a given request uses one

	stickyUA bool   // if true, every request from this worker reuses one User-Agent
	fixedUA  string // the User-Agent picked once, reused (set when stickyUA is true)

	fixedIP string // attack mode: pin every request to one IP

	floodCampaign string // distributed fraud: hammer one campaign ID from many IPs
	jitterMaxMs   int    // organic-looking random delay added on top of the base interval
}

func randomEvent(rng *rand.Rand, p *profile, isBlacklisted *bool) ClickEvent {
	ip := p.fixedIP
	if ip == "" {
		// occasionally inject a real blacklisted IP instead of a random one
		if len(p.blacklistIPs) > 0 && rng.Float64() < p.blacklistChance {
			ip = p.blacklistIPs[rng.Intn(len(p.blacklistIPs))]
			*isBlacklisted = true
		} else {
			ip = randomIP(rng)
			*isBlacklisted = false
		}
	}

	ua := userAgents[rng.Intn(len(userAgents))]
	if p.stickyUA {
		ua = p.fixedUA
	}

	campaign := campaignIDs[rng.Intn(len(campaignIDs))]
	if p.floodCampaign != "" {
		campaign = p.floodCampaign
	}

	return ClickEvent{
		IP:         ip,
		UserAgent:  ua,
		CampaignID: campaign,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// --- counters ---
// Separate atomic counters per outcome so the final report can show
// exactly which defense layer caught each blocked request — this is
// what lets QA cross-check these numbers against the dashboard.

type counters struct {
	sent        atomic.Int64 // every request attempted
	ok          atomic.Int64 // HTTP 200–299 (clean click, passed through)
	blocked     atomic.Int64 // HTTP 429 (Redis rate limiter)
	blacklisted atomic.Int64 // HTTP 403 (Bloom filter blacklist)
	errs        atomic.Int64 // network errors or unexpected status codes
}

// --- worker ---

func worker(
	target string,
	client *http.Client,
	rng *rand.Rand,
	p *profile,
	rps int,
	deadline time.Time,
	c *counters,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	interval := time.Second / time.Duration(rps)

	for time.Now().Before(deadline) {
		start := time.Now()

		var injectedBlacklist bool
		event := randomEvent(rng, p, &injectedBlacklist)
		body, _ := json.Marshal(event)

		// The engine identifies the caller via the X-Forwarded-For
		// header, not the "ip" field in the JSON body — see
		// getClientIP() in cmd/engine/main.go.
		req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			c.sent.Add(1)
			c.errs.Add(1)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", event.IP)

		resp, err := client.Do(req)
		c.sent.Add(1)

		if err != nil {
			c.errs.Add(1)
		} else {
			resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				c.ok.Add(1)
			case http.StatusTooManyRequests: // 429 — Redis rate limiter
				c.blocked.Add(1)
			case http.StatusForbidden: // 403 — Bloom filter blacklist
				c.blacklisted.Add(1)
			default:
				c.errs.Add(1)
			}
		}

		// Base pacing to hit the target rps, optionally with extra
		// random jitter layered on top to mimic a "smart" bot that
		// avoids a perfectly mechanical, easily-fingerprinted rhythm.
		elapsed := time.Since(start)
		sleep := interval - elapsed
		if p.jitterMaxMs > 0 {
			sleep += time.Duration(rng.Intn(p.jitterMaxMs)) * time.Millisecond
		}
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

// --- report ---

func printReport(target string, mode string, c *counters, elapsed float64) {
	total := c.sent.Load()
	ok := c.ok.Load()
	blocked := c.blocked.Load()
	blacklisted := c.blacklisted.Load()
	errs := c.errs.Load()

	caught := blocked + blacklisted
	var efficiency float64
	if total > 0 {
		efficiency = float64(caught) / float64(total) * 100
	}

	fmt.Println()
	fmt.Println("==============================")
	fmt.Println("   ANTI-FRAUD TEST REPORT")
	fmt.Println("==============================")
	fmt.Printf("  Target URL          : %s\n", target)
	fmt.Printf("  Traffic Mode        : %s\n", mode)
	fmt.Printf("  Duration            : %.1fs\n", elapsed)
	fmt.Println("------------------------------")
	fmt.Printf("  Total Requests Sent : %d\n", total)
	fmt.Printf("  Clean Clicks (200)  : %d\n", ok)
	fmt.Printf("  Rate-Limit Hits(429): %d\n", blocked)
	fmt.Printf("  Blacklist Hits (403): %d\n", blacklisted)
	fmt.Printf("  Errors (other)      : %d\n", errs)
	fmt.Println("------------------------------")
	fmt.Printf("  Overall Catch Rate  : %.1f%%\n", efficiency)
	fmt.Println("==============================")
}

// --- main ---

func main() {
	target := flag.String("target", "http://localhost:8080/v1/click", "Engine endpoint")
	workers := flag.Int("workers", 10, "Number of concurrent goroutines")
	rps := flag.Int("rps", 10, "Requests per second per worker")
	dur := flag.Duration("duration", 30*time.Second, "How long to run (e.g. 30s, 2m)")

	attack := flag.Bool("attack", false, "Attack mode: one fixed IP at 100 rps per worker")
	attackIP := flag.String("attack-ip", "1.2.3.4", "Fixed IP used in attack mode")

	blacklistFile := flag.String("blacklist-file", "deployments/blacklists/dirty_ips.txt", "Path to known-bad IP list")
	blacklistChance := flag.Float64("blacklist-chance", 0.0, "Probability (0.0-1.0) a request uses a real blacklisted IP")

	stickyUA := flag.Bool("sticky-ua", false, "Distributed fraud: keep one User-Agent across many IPs")
	floodCampaign := flag.String("flood-campaign", "", "Distributed fraud: hammer this single campaign ID from many IPs")
	jitterMs := flag.Int("jitter-ms", 0, "Add up to this many ms of random delay per request (mimics organic timing)")

	flag.Parse()

	p := &profile{
		blacklistChance: *blacklistChance,
		stickyUA:        *stickyUA,
		floodCampaign:   *floodCampaign,
		jitterMaxMs:     *jitterMs,
	}

	// load real blacklisted IPs if requested
	if *blacklistChance > 0 {
		ips, err := loadBlacklistIPs(*blacklistFile)
		if err != nil {
			fmt.Printf("⚠️  Could not load blacklist file (%v) — continuing without blacklist injection\n", err)
		} else {
			p.blacklistIPs = ips
			fmt.Printf("Loaded %d blacklisted IPs from %s\n", len(ips), *blacklistFile)
		}
	}

	mode := "NORMAL"
	if *attack {
		mode = "ATTACK"
		*rps = 100
		p.fixedIP = *attackIP
	}
	if *stickyUA {
		mode += "+STICKY_UA"
	}
	if *floodCampaign != "" {
		mode += "+CAMPAIGN_FLOOD"
	}
	if *blacklistChance > 0 {
		mode += "+BLACKLIST_INJECT"
	}

	fmt.Printf("🔀 MODE: %s — %d workers × %d rps = %d rps total\n", mode, *workers, *rps, *workers**rps)
	fmt.Printf("   target:   %s\n", *target)
	fmt.Printf("   duration: %s\n\n", *dur)

	client := &http.Client{Timeout: 5 * time.Second}

	var c counters
	var wg sync.WaitGroup

	deadline := time.Now().Add(*dur)
	start := time.Now()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))

		// Each worker gets its own copy of the profile so that
		// stickyUA can pick one User-Agent PER WORKER (simulating
		// many distinct bot instances, each consistent with itself
		// but different from the others) rather than one UA shared
		// globally across all workers.
		workerProfile := *p
		if p.stickyUA {
			workerProfile.fixedUA = userAgents[rng.Intn(len(userAgents))]
		}

		go worker(*target, client, rng, &workerProfile, *rps, deadline, &c, &wg)
	}

	ticker := time.NewTicker(time.Second)
	go func() {
		prev := int64(0)
		for range ticker.C {
			cur := c.sent.Load()
			fmt.Printf("\r  sent: %-8d  ok: %-8d  429: %-8d  403: %-8d  rps: %-6d",
				cur, c.ok.Load(), c.blocked.Load(), c.blacklisted.Load(), cur-prev)
			prev = cur
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		ticker.Stop()
		printReport(*target, mode, &c, time.Since(start).Seconds())
		os.Exit(0)
	}()

	wg.Wait()
	ticker.Stop()

	printReport(*target, mode, &c, time.Since(start).Seconds())
}
