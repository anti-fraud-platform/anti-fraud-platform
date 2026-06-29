package bloom

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestIPFilter_LogicAndMemory(t *testing.T) {
	if err := os.Chdir("../../"); err != nil {
		t.Fatalf("failed to change directory to project root: %v", err)
	}

	filePath := "deployments/blacklists/dirty_ips.txt"
	goodIP := "10.0.0.1"

	// open file to extract a bad IP and ensure goodIP is not blacklisted
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open blacklist file: %v", err)
	}
	defer file.Close()

	var badIP string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip == "" {
			continue
		}
		if ip == goodIP {
			t.Fatalf("test config error: goodIP %s is found in the blacklist file", goodIP)
		}
		if badIP == "" {
			badIP = ip
		}
	}

	if badIP == "" {
		t.Fatalf("blacklist file is empty, cannot proceed with tests")
	}

	// initialize the filter
	filter, err := NewIPFilter(filePath)
	if err != nil {
		t.Fatalf("failed to initialize IPFilter: %v", err)
	}

	// 1. logic check: Known-bad must be true, known-good must be false
	if !filter.IsBlacklisted(badIP) {
		t.Errorf("expected IP %s to be blacklisted, but it wasn't", badIP)
	}
	if filter.IsBlacklisted(goodIP) {
		t.Errorf("expected clean IP %s to pass, but it triggered a false positive", goodIP)
	}

	// 2. memory sanity check
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	iterations := 5000
	for i := 0; i < iterations; i++ {
		testIP := fmt.Sprintf("185.220.101.%d", i%255)
		_ = filter.IsBlacklisted(testIP)
	}

	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memoryGrowth := int64(memAfter.HeapAlloc) - int64(memBefore.HeapAlloc)
	if memoryGrowth < 0 {
		memoryGrowth = 0
	}

	// threshold: 512kb for test loop overhead
	if memoryGrowth > 512*1024 {
		t.Errorf("memory leak detected: allocated %d bytes during lookups", memoryGrowth)
	} else {
		t.Logf("memory check passed: growth %d bytes", memoryGrowth)
	}
}