# Local Setup

This guide is for a clean clone on a new machine. It assumes only Git, Go, and Docker.

## 1. Install prerequisites

Install:

- Go `1.26.x`
- Docker Desktop or Docker Engine
- Docker Compose support
- Git

Then verify the tools:

```bash
go version
docker --version
docker compose version
docker-compose version
git --version
```

You do not need both Compose commands. One is enough. Use the one your machine actually supports.

## 2. Clone the repository

```bash
git clone git@github.com:anti-fraud-platform/anti-fraud-platform.git
cd anti-fraud-platform
```

If SSH access to GitHub is not configured yet, use HTTPS instead:

```bash
git clone https://github.com/anti-fraud-platform/anti-fraud-platform.git
cd anti-fraud-platform
```

## 3. Build and start the full stack

Start everything in the background:

```bash
docker compose up --build -d
```

If your installation uses the legacy command form:

```bash
docker-compose up --build -d
```

This should build and start six services:

- `antifraud-engine`
- `antifraud-nginx-engine`
- `antifraud-analytics`
- `antifraud-frontend`
- `antifraud-postgres`
- `antifraud-redis`

For real GeoIP checks, place the real MaxMind databases at `geoip/GeoLite2-Country.mmdb`, `geoip/GeoLite2-City.mmdb`, and `geoip/GeoLite2-ASN.mmdb` before you begin testing. The short version is in [geoip/README.md](../geoip/README.md).

The stack still starts without them, but GeoIP verification is incomplete until you add them.

If your Postgres volume already existed from an older version of the project, the services now run an idempotent schema upgrade on startup. That means older local data can stay in place while missing tables and columns are added automatically.

## 4. Confirm that the services are up

List containers:

```bash
docker compose ps
```

Expected state:

- `engine` is `Up`
- `nginx_engine` is `Up`
- `analytics` is `Up`
- `frontend` is `Up`
- `postgres` is `Up (healthy)`
- `redis` is `Up (healthy)`

Expected host ports:

- `3001` -> frontend
- `8082` -> analytics
- `9090` -> nginx reverse proxy for the engine
- `5433` -> postgres
- `6380` -> redis

The engine is internal-only now. It is reachable inside Compose as `engine:8080`, but it is not published directly to the host.

## 5. Verify the stack from the browser and terminal

Open these URLs:

- Dashboard: `http://localhost:3001`
- Click simulator: `http://localhost:9090`
- Analytics JSON: `http://localhost:8082/v1/analytics/stats`

Then verify from the terminal:

```bash
curl http://localhost:8082/v1/analytics/stats
```

```bash
curl http://localhost:9090/v1/challenge
```

That challenge endpoint should return JSON with `challenge_id` and `nonce`.

Now send a click without solving the challenge:

```bash
curl -X POST http://localhost:9090/click \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":"demo_campaign"}'
```

Expected result: a JSON response with `status: "flagged"`.
That is the correct behavior. A raw curl request does not solve the JS challenge first, so the engine should not treat it as a clean human click.

If you want to see a real `success` response, open `http://localhost:9090` and click `Send Real Click`. That page fetches `/v1/challenge`, solves it in the browser, waits briefly, and then submits the click.

To confirm the engine is intentionally not exposed directly, this should fail:

```bash
curl http://localhost:8080/v1/challenge
```

## 6. Common local issues

### Compose command mismatch

If `docker compose` says `unknown command`, use:

```bash
docker-compose up --build -d
```

### Port conflicts

If one of these ports is already taken, stop the conflicting process or change the host-side port in `docker-compose.yml`:

- `3001`
- `8082`
- `9090`
- `5433`
- `6380`

### Download failures

If image pulls or dependency downloads fail, check the local Docker and package-manager setup first. Those problems are usually environmental, not a repository bug.

### Docker daemon access

If Compose cannot talk to Docker on macOS, check whether Docker Desktop or Colima is running and whether your Docker context points at the right socket.

### Old nginx container still caching the engine IP

The permanent fix is already in `deployments/nginx/engine.conf`: nginx now resolves `engine` through Docker DNS (`127.0.0.11`) instead of caching a single container IP at startup.

This does not mean the project runs its own DNS server. Docker already provides an internal DNS service for every container on the Compose network. That is what lets services call each other by name, for example `engine`, `postgres`, `redis`, or `analytics`, instead of hardcoded IP addresses.

The problem was specific to nginx. After `docker compose up --build`, the `engine` container could be recreated with a new internal IP address, while `nginx_engine` kept running with the old cached upstream address. Port `9090` then broke until nginx was restarted. The fix was to make nginx re-resolve the hostname `engine` through Docker's built-in DNS each time it needs a fresh upstream address, instead of holding on to one stale container IP.

If you are updating an older running environment that still has the previous nginx config loaded, run this once after deploy:

```bash
docker compose restart nginx_engine
```

That one restart is only a rollout safety step for older containers. Fresh builds from the current repo should not need a manual nginx bounce after every engine rebuild.

## 7. Real GeoIP verification

1. Download and extract `GeoLite2-Country.mmdb`, `GeoLite2-City.mmdb`, and `GeoLite2-ASN.mmdb`.
2. Place them in `geoip/`.
3. Rebuild the engine:

```bash
docker compose up --build -d engine
```

4. Run a real lookup:

```bash
go run ./cmd/geoiplookup -ip 8.8.8.8
```

That command should print JSON with country, city, and ASN fields.

For the full request-to-database manual check, run:

```bash
bash scripts/geoip/e2e_real_ip.sh
```

## 8. Clean rerun from scratch

If you want to repeat the setup with fresh containers but keep the existing Postgres volume:

```bash
docker compose down
docker compose up --build -d
```

If you want to test first-run initialization again, including database bootstrap, remove the named volume too:

```bash
docker compose down -v
docker volume rm antifraud_pg_data
docker compose up --build -d
```

Use the destructive version only when you intentionally want a clean database.
