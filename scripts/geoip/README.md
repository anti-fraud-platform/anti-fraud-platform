# GeoIP Manual Checks

This folder is for local and VM-side manual verification of the GeoIP pipeline.

## What `e2e_real_ip.sh` does

It validates the full path:

1. Reads Country, City, and ASN directly from the three local MaxMind databases.
2. Fetches a real JS challenge from the engine.
3. Sends a click through nginx with a public IP in `X-Forwarded-For`.
4. Waits for the batch logger to flush the row.
5. Queries `click_logs` from Postgres.
6. Compares the stored `country`, `city`, `asn_number`, and `asn_org` against the direct MaxMind lookup.

## Usage

```bash
bash scripts/geoip/e2e_real_ip.sh
```

Use a different public IP if needed:

```bash
bash scripts/geoip/e2e_real_ip.sh 1.1.1.1
```