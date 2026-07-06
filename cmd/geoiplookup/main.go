package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"

	"anti-fraud/internal/geoiputil"
)

func main() {
	countryDBPath := flag.String("country-db", "./geoip/GeoLite2-Country.mmdb", "path to GeoLite2-Country.mmdb")
	cityDBPath := flag.String("city-db", "./geoip/GeoLite2-City.mmdb", "path to GeoLite2-City.mmdb")
	asnDBPath := flag.String("asn-db", "./geoip/GeoLite2-ASN.mmdb", "path to GeoLite2-ASN.mmdb")
	ipValue := flag.String("ip", "", "public IP address to resolve")
	flag.Parse()

	if *ipValue == "" {
		log.Fatal("missing required -ip flag")
	}

	parsedIP := net.ParseIP(*ipValue)
	if parsedIP == nil {
		log.Fatalf("invalid IP address: %s", *ipValue)
	}

	resolver, errs := geoiputil.OpenBestEffort(geoiputil.Paths{
		Country: *countryDBPath,
		City:    *cityDBPath,
		ASN:     *asnDBPath,
	})
	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("failed to open GeoIP database: %v", err)
		}
		log.Fatal("geoiplookup requires all requested GeoIP databases to be readable")
	}
	defer resolver.Close()

	result := resolver.Lookup(parsedIP)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		log.Fatalf("failed to write JSON output: %v", err)
	}
}
