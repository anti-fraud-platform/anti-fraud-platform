# GeoIP Database Setup

The repository stores the real MaxMind databases directly in `geoip/`.

These files should be present:

```text
geoip/GeoLite2-Country.mmdb
geoip/GeoLite2-City.mmdb
geoip/GeoLite2-ASN.mmdb
```

The engine reads them from:

```text
/usr/share/GeoIP/GeoLite2-Country.mmdb
/usr/share/GeoIP/GeoLite2-City.mmdb
/usr/share/GeoIP/GeoLite2-ASN.mmdb
```

Those paths are wired through the `GEOIP_COUNTRY_DB_PATH`, `GEOIP_CITY_DB_PATH`, and `GEOIP_ASN_DB_PATH` environment variables in `docker-compose.yml`.

## How to refresh the databases

1. Replace the three `.mmdb` files in this directory with newer MaxMind versions.
2. Rebuild the engine:

```bash
docker compose up --build -d engine
```

## How to verify direct lookups from all three databases

Run the helper command with a real public IP:

```bash
go run ./cmd/geoiplookup -ip 8.8.8.8
```

Example output:

```json
{
  "ip": "8.8.8.8",
  "country_iso": "US",
  "country_name": "United States",
  "city_name": "",
  "subdivision_name": "",
  "asn_number": 15169,
  "asn_org": "Google LLC"
}
```

Use a public IP that fits your own test case. The important part is that the lookup comes from real `.mmdb` files, not from generated test data.

## Full end-to-end manual check

To verify the whole request path through nginx, engine, batch logging, and Postgres:

```bash
bash scripts/geoip/e2e_real_ip.sh
```
