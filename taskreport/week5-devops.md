# Week 5 DevOps Report

## What I changed this week

This week I worked on the infrastructure and verification side of the project rather than adding a new fraud rule.

### 1. Nginx stale-upstream fix

The first problem was nginx.

We hit the same issue several times: after `docker compose up --build`, the `engine` container could be recreated with a new internal IP address, but `nginx_engine` stayed alive and kept sending requests to the old address. When that happened, port `9090` stopped working until nginx was restarted by hand.

I fixed this in `deployments/nginx/engine.conf`.

Before the change, nginx was effectively tied to one cached upstream IP. After the change, nginx resolves `engine` through Docker's built-in DNS at `127.0.0.11`. Docker already provides internal DNS for all containers on the Compose network, so services can talk to each other by service name such as `engine`, `postgres`, `redis`, or `analytics`. The goal here was not to add a new DNS server to the project. The goal was to make nginx use Docker DNS correctly so it can recover when the `engine` container gets recreated.

I also kept one rollout safety step in the GitLab deploy job:

```bash
docker-compose restart nginx_engine
```

Fresh environments should not need this on every deploy anymore. It is still useful once for older running environments that were started before the nginx fix.

### 2. CI improvements

The second area was CI.

The previous GitHub Actions workflow mainly answered one question: does the code compile? It did not answer a more important question: does the full stack still behave correctly after changes?

I kept the existing backend and frontend checks, then added the missing parts:

- frontend build artifact upload
- `govulncheck`
- a real integration stage based on Docker Compose

The integration stage now boots the stack and runs smoke checks against the running services.

I also split the smoke checks into small scripts under `scripts/ci/`. Each script proves one thing:

1. the stack becomes reachable
2. the frontend shell loads
3. the simulator page on `9090` loads
4. analytics returns the expected response fields
5. `/v1/challenge` returns a real challenge payload
6. a click without a solved challenge is flagged
7. nginx still reaches the engine after recreating only the `engine` container

This made the CI logic easier to read and easier to debug. When one check fails, the failing script already points to the area that needs attention.

### 3. GeoIP-only detection

The third area was GeoIP.

I extended the local GeoIP path so it uses all three MaxMind databases stored in `geoip/`:

- Country
- City
- ASN

To support that, I added a shared resolver in `internal/geoiputil/resolver.go`.

I then updated the batch logger so each click log can be enriched with:

- `country`
- `city`
- `asn_number`
- `asn_org`

I also extended the database schema so these fields are stored in `click_logs`.

After that, I removed the old file-based `dirty_ips` path from the runtime flow and switched the engine to GeoIP-only blocking.

The engine now reads blocking rules from environment variables:

- `GEOIP_BLOCKED_COUNTRIES`
- `GEOIP_BLOCKED_ASN_NUMBERS`
- `GEOIP_BLOCKED_ASN_KEYWORDS`

For the default local stack, I configured ASN keyword rules that catch traffic coming from common hosting and proxy networks. That means the hard block layer is now based on MaxMind country / ASN data rather than a hand-maintained IP text file.

### 4. Safe schema upgrades for old volumes

I also wanted to avoid breaking older local databases.

Some teammates already had an existing `antifraud_pg_data` volume from earlier weeks. In that situation, `docker-entrypoint-initdb.d` does not run again, so new tables and columns would normally be missing.

To solve that, I added an idempotent schema upgrade step that runs on service startup. Both `engine` and `analytics` now apply the current schema automatically. That means an older local volume can stay in place while the missing tables and columns are added in the background.

### 5. Manual GeoIP verification flow

Finally, I added a manual GeoIP verification path.

The helper `cmd/geoiplookup` now reads all three `.mmdb` files and prints one combined lookup result.

On top of that, `scripts/geoip/e2e_real_ip.sh` runs a full manual end-to-end check:

1. it performs a direct MaxMind lookup for a public IP
2. it fetches a real JS challenge from the engine
3. it sends a click through nginx with `X-Forwarded-For`
4. it waits for the batch logger to flush the row
5. it reads the stored row back from Postgres
6. it compares the stored GeoIP fields with the direct lookup result

This gives us a way to verify not only that the `.mmdb` files are readable, but also that the full path from request to database enrichment works as expected.

## Files added or updated

Main config and runtime changes:

- `deployments/nginx/engine.conf`
- `.github/workflows/ci.yml`
- `.gitlab-ci.yml`
- `docker-compose.yml`
- `internal/dbschema/schema.go`
- `internal/geoiputil/resolver.go`
- `internal/logger/batch_logger.go`
- `cmd/engine/main.go`
- `cmd/analytics/main.go`
- `deployments/init-db.sql`

Verification and helper scripts:

- `scripts/ci/compose_smoke.sh`
- `scripts/ci/lib/common.sh`
- `scripts/ci/checks/*`
- `scripts/ci/README.md`
- `cmd/geoiplookup/main.go`
- `scripts/geoip/e2e_real_ip.sh`
- `scripts/geoip/lib/common.sh`
- `scripts/geoip/README.md`

Documentation updates:

- `README.md`
- `docs/SETUP.md`
- `geoip/README.md`

## What this means for the system

The platform still has the same core responsibilities:

1. receive click traffic
2. decide whether the click looks human or suspicious
3. store enough data for analytics and later investigation

This week the main improvement was not a new detection rule. It was making the platform more reliable and easier to prove:

- nginx is less fragile after container rebuilds
- CI checks real stack behavior, not only compilation
- GeoIP data is stored in the database instead of staying outside the pipeline
- old databases can move forward without a manual reset

## Suggested verification commands

```bash
docker-compose up --build -d
docker-compose ps
bash scripts/ci/compose_smoke.sh
go run ./cmd/geoiplookup -ip 8.8.8.8
bash scripts/geoip/e2e_real_ip.sh
```
