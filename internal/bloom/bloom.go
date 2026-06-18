package bloom

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/bits-and-blooms/bloom/v3"
)

type IPFilter struct {
	filter *bloom.BloomFilter
}

func NewIPFilter(filePath string) (*IPFilter, error) {
	bf := bloom.NewWithEstimates(15000, 0.01)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" {
			bf.Add([]byte(ip))
			count++
		}
	}

	log.Printf("[BloomFilter] Successfully loaded %d bad IPs into memory", count)
	return &IPFilter{filter: bf}, nil
}

func (f *IPFilter) IsBlacklisted(ip string) bool {
	return f.filter.Test([]byte(ip))
}
