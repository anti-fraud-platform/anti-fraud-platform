package geoiputil

import (
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
)

type Paths struct {
	Country string
	City    string
	ASN     string
}

type LookupResult struct {
	IP          string `json:"ip"`
	CountryISO  string `json:"country_iso"`
	CountryName string `json:"country_name"`
	CityName    string `json:"city_name"`
	Subdivision string `json:"subdivision_name"`
	ASNNumber   uint   `json:"asn_number"`
	ASNOrg      string `json:"asn_org"`
}

type Resolver struct {
	countryReader *geoip2.Reader
	cityReader    *geoip2.Reader
	asnReader     *geoip2.Reader
}

func (r *Resolver) HasCountry() bool {
	return r != nil && r.countryReader != nil
}

func (r *Resolver) HasCity() bool {
	return r != nil && r.cityReader != nil
}

func (r *Resolver) HasASN() bool {
	return r != nil && r.asnReader != nil
}

func (r *Resolver) HasAny() bool {
	return r.HasCountry() || r.HasCity() || r.HasASN()
}

func PathsFromEnv() Paths {
	countryPath := os.Getenv("GEOIP_COUNTRY_DB_PATH")
	if countryPath == "" {
		countryPath = os.Getenv("GEOIP_DB_PATH")
	}
	if countryPath == "" {
		countryPath = "/usr/share/GeoIP/GeoLite2-Country.mmdb"
	}

	cityPath := os.Getenv("GEOIP_CITY_DB_PATH")
	if cityPath == "" {
		cityPath = "/usr/share/GeoIP/GeoLite2-City.mmdb"
	}

	asnPath := os.Getenv("GEOIP_ASN_DB_PATH")
	if asnPath == "" {
		asnPath = "/usr/share/GeoIP/GeoLite2-ASN.mmdb"
	}

	return Paths{
		Country: countryPath,
		City:    cityPath,
		ASN:     asnPath,
	}
}

func OpenBestEffort(paths Paths) (*Resolver, []error) {
	resolver := &Resolver{}
	var errs []error

	if paths.Country != "" {
		reader, err := geoip2.Open(paths.Country)
		if err != nil {
			errs = append(errs, err)
		} else {
			resolver.countryReader = reader
		}
	}

	if paths.City != "" {
		reader, err := geoip2.Open(paths.City)
		if err != nil {
			errs = append(errs, err)
		} else {
			resolver.cityReader = reader
		}
	}

	if paths.ASN != "" {
		reader, err := geoip2.Open(paths.ASN)
		if err != nil {
			errs = append(errs, err)
		} else {
			resolver.asnReader = reader
		}
	}

	return resolver, errs
}

func (r *Resolver) Lookup(ip net.IP) LookupResult {
	result := LookupResult{}
	if ip == nil {
		return result
	}
	result.IP = ip.String()

	if r.countryReader != nil {
		if record, err := r.countryReader.Country(ip); err == nil && record != nil {
			result.CountryISO = record.Country.IsoCode
			result.CountryName = record.Country.Names["en"]
		}
	}

	if r.cityReader != nil {
		if record, err := r.cityReader.City(ip); err == nil && record != nil {
			result.CityName = record.City.Names["en"]
			if len(record.Subdivisions) > 0 {
				result.Subdivision = record.Subdivisions[0].Names["en"]
			}
			if result.CountryISO == "" {
				result.CountryISO = record.Country.IsoCode
			}
			if result.CountryName == "" {
				result.CountryName = record.Country.Names["en"]
			}
		}
	}

	if r.asnReader != nil {
		if record, err := r.asnReader.ASN(ip); err == nil && record != nil {
			result.ASNNumber = record.AutonomousSystemNumber
			result.ASNOrg = record.AutonomousSystemOrganization
		}
	}

	return result
}

func (r *Resolver) Close() error {
	if r.countryReader != nil {
		_ = r.countryReader.Close()
	}
	if r.cityReader != nil {
		_ = r.cityReader.Close()
	}
	if r.asnReader != nil {
		_ = r.asnReader.Close()
	}
	return nil
}
