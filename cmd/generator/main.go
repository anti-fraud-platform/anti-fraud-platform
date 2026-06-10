package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ClickEvent mirrors the payload expected by POST /v1/click
type ClickEvent struct {
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	CampaignID string `json:"campaign_id"`
	Timestamp  int64  `json:"timestamp"`
}

// random data pools 

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

//  worker 

func worker(
	id int,
	target string,
	client *http.Client,
	rng *rand.Rand,
	fixedIP string,
	rps int,
	duration time.Duration,
	sent *atomic.Int64,
	ok *atomic.Int64,
	errs *atomic.Int64,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	interval := time.Second / time.Duration(rps) // gap between requests
	deadline := time.Now().Add(duration)

	for time.Now().Before(deadline) {
		start := time.Now()

		event := randomEvent(rng, fixedIP)
		body, _ := json.Marshal(event)

		resp, err := client.Post(target, "application/json", bytes.NewReader(body))
		sent.Add(1)

		if err != nil {
			errs.Add(1)
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				ok.Add(1)
			} else {
				errs.Add(1)
			}
		}

		// honour the requested rate — sleep only for the remainder of the interval
		elapsed := time.Since(start)
		if sleep := interval - elapsed; sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

func main() {
	//  flags 
	target := flag.String("target", "http://localhost:8080/v1/click", "Engine endpoint")
	workers := flag.Int("workers", 10, "Number of concurrent goroutines")
	rps := flag.Int("rps", 10, "Requests per second per worker")
	dur := flag.Duration("duration", 30*time.Second, "How long to run (e.g. 30s, 2m)")
	attack := flag.Bool("attack", false, "Attack mode: single IP hammers at 100 rps per worker")
	attackIP := flag.String("attack-ip", "1.2.3.4", "IP to use in attack mode")
	flag.Parse()

	// attack mode overrides rps and fixes the source IP
	fixedIP := ""
	if *attack {
		*rps = 100
		fixedIP = *attackIP
		fmt.Printf("⚠️  ATTACK MODE — IP=%s, %d workers × %d rps = %d rps total\n",
			fixedIP, *workers, *rps, *workers**rps)
	} else {
		fmt.Printf("🔀 NORMAL MODE — %d workers × %d rps = %d rps total\n",
			*workers, *rps, *workers**rps)
	}

	fmt.Printf("   target:   %s\n", *target)
	fmt.Printf("   duration: %s\n\n", *dur)

	// shared HTTP client — keep-alives on, generous timeout
	client := &http.Client{Timeout: 5 * time.Second}

	var (
		sent, okCount, errCount atomic.Int64
		wg                      sync.WaitGroup
	)

	start := time.Now()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		// each worker gets its own RNG so there's no lock contention
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go worker(i, *target, client, rng, fixedIP, *rps, *dur, &sent, &okCount, &errCount, &wg)
	}

	// live progress ticker
	ticker := time.NewTicker(time.Second)
	go func() {
		prev := int64(0)
		for range ticker.C {
			cur := sent.Load()
			fmt.Printf("\r  sent: %-8d  ok: %-8d  errors: %-8d  current rps: %-6d",
				cur, okCount.Load(), errCount.Load(), cur-prev)
			prev = cur
		}
	}()

	wg.Wait()
	ticker.Stop()

	elapsed := time.Since(start).Seconds()
	total := sent.Load()

	fmt.Printf("\n\n=== Done ===\n")
	fmt.Printf("  Total sent : %d\n", total)
	fmt.Printf("  OK (2xx)   : %d\n", okCount.Load())
	fmt.Printf("  Errors     : %d\n", errCount.Load())
	fmt.Printf("  Duration   : %.1fs\n", elapsed)
	fmt.Printf("  Avg RPS    : %.0f\n", float64(total)/elapsed)
}