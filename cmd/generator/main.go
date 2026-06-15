package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
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

//  random data pools

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

func randomEvent(rng *rand.Rand, fixedIP string) ClickEvent {
	ip := fixedIP
	if ip == "" {
		ip = randomIP(rng)
	}
	return ClickEvent{
		IP:         ip,
		UserAgent:  userAgents[rng.Intn(len(userAgents))],
		CampaignID: campaignIDs[rng.Intn(len(campaignIDs))],
		Timestamp:  time.Now().UnixMilli(),
	}
}

//  counters
// Four separate atomic counters so the report is precise.
// We track 429 separately from generic errors because the task
// specifically wants to show "blocked by rate limiter" vs real failures.

type counters struct {
	sent    atomic.Int64 // every request attempted
	ok      atomic.Int64 // HTTP 200–299
	blocked atomic.Int64 // HTTP 429 (rate limited)
	errs    atomic.Int64 // network errors or unexpected status codes
}

//  worker

func worker(
	target string,
	client *http.Client,
	rng *rand.Rand,
	fixedIP string,
	rps int,
	deadline time.Time,
	c *counters,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	interval := time.Second / time.Duration(rps)

	for time.Now().Before(deadline) {
		start := time.Now()

		event := randomEvent(rng, fixedIP)
		body, _ := json.Marshal(event)

		req, err := http.NewRequest("POST", target, bytes.NewReader(body))
		if err != nil {
			c.errs.Add(1)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", event.IP)
		req.Header.Set("X-Real-IP", event.IP)

		resp, err := client.Do(req)
		c.sent.Add(1)

		if err != nil {
			c.errs.Add(1)
		} else {
			resp.Body.Close()
			switch {
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				c.ok.Add(1)
			case resp.StatusCode == http.StatusTooManyRequests: // 429
				c.blocked.Add(1)
			default:
				c.errs.Add(1)
			}
		}

		if sleep := interval - time.Since(start); sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

//  report

func printReport(target string, c *counters, elapsed float64) {
	total := c.sent.Load()
	ok := c.ok.Load()
	blocked := c.blocked.Load()
	errs := c.errs.Load()

	var efficiency float64
	if total > 0 {
		efficiency = float64(blocked) / float64(total) * 100
	}

	fmt.Println()
	fmt.Println("==============================")
	fmt.Println("   ANTI-FRAUD TEST REPORT")
	fmt.Println("==============================")
	fmt.Printf("  Target URL         : %s\n", target)
	fmt.Printf("  Duration           : %.1fs\n", elapsed)
	fmt.Println("------------------------------")
	fmt.Printf("  Total Requests Sent: %d\n", total)
	fmt.Printf("  Success  (200 OK)  : %d\n", ok)
	fmt.Printf("  Blocked  (429)     : %d\n", blocked)
	fmt.Printf("  Errors   (other)   : %d\n", errs)
	fmt.Println("------------------------------")
	fmt.Printf("  Block Rate         : %.1f%%\n", efficiency)
	fmt.Println("==============================")
}

//  main

func main() {
	target := flag.String("target", "http://localhost:8080/v1/click", "Engine endpoint")
	workers := flag.Int("workers", 10, "Number of concurrent goroutines")
	rps := flag.Int("rps", 10, "Requests per second per worker")
	dur := flag.Duration("duration", 30*time.Second, "How long to run (e.g. 30s, 2m)")
	attack := flag.Bool("attack", false, "Attack mode: one IP at 100 rps per worker")
	attackIP := flag.String("attack-ip", "1.2.3.4", "Fixed IP used in attack mode")
	flag.Parse()

	fixedIP := ""
	if *attack {
		*rps = 100
		fixedIP = *attackIP
		fmt.Printf("⚠️  ATTACK MODE — IP=%s | %d workers × %d rps = %d rps total\n",
			fixedIP, *workers, *rps, *workers**rps)
	} else {
		fmt.Printf("🔀 NORMAL MODE — %d workers × %d rps = %d rps total\n",
			*workers, *rps, *workers**rps)
	}
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
		go worker(*target, client, rng, fixedIP, *rps, deadline, &c, &wg)
	}

	// live progress line — updates every second
	ticker := time.NewTicker(time.Second)
	go func() {
		prev := int64(0)
		for range ticker.C {
			cur := c.sent.Load()
			fmt.Printf("\r  sent: %-8d  ok: %-8d  blocked(429): %-8d  rps: %-6d",
				cur, c.ok.Load(), c.blocked.Load(), cur-prev)
			prev = cur
		}
	}()

	// handle Ctrl+C — print report before exiting
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		ticker.Stop()
		printReport(*target, &c, time.Since(start).Seconds())
		os.Exit(0)
	}()

	wg.Wait()
	ticker.Stop()

	printReport(*target, &c, time.Since(start).Seconds())
}
