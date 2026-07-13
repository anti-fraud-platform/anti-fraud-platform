# Week 6 DevOps Report

## What I implemented

This week I added a real monitoring path and a reproducible load-test path for the deployed stack.

### Monitoring

I added a new internal package, `internal/observability`, and wired it into both Go services.

For `engine` and `analytics`, I added:

- `/metrics` for Prometheus scraping
- HTTP middleware that records request count and request duration
- live dependency gauges for PostgreSQL and Redis

The most important custom metrics are:

- `antifraud_http_requests_total`
- `antifraud_http_request_duration_seconds`
- `antifraud_dependency_up`

This means Prometheus now collects:

- real request rate from the application itself
- response code distribution by endpoint
- Go runtime metrics like `go_goroutines`
- dependency health from the point of view of the running service

### Docker Compose monitoring profile

I added a `monitoring` profile to `docker-compose.yml`.

It includes:

- `prometheus`
- `grafana`
- `node_exporter`

I kept it behind a Compose profile so the normal CI smoke stack stays lightweight and does not need host-level node-exporter mounts.

### Prometheus and Grafana config

I added provisioned config under `deployments/monitoring/`.

Prometheus scrapes:

- itself
- `engine:8080/metrics`
- `analytics:8081/metrics`
- `node_exporter:9100`

Grafana is preloaded with one dashboard:

- `Anti-Fraud Platform Overview`

That dashboard shows:

- click req/s
- `200 / 403 / 429` over time
- p95 click latency
- engine goroutine count
- node CPU and memory usage
- service-level Redis / PostgreSQL health

### Load test path

I added a dedicated `scripts/loadtest/` folder.

There are two k6 scenarios:

1. `k6_real_click_ramp.js`
   Uses the real `/v1/challenge` -> `/click` flow and ramps load until latency or errors rise.
2. `k6_status_mix.js`
   Generates a mixed stream of successful, rate-limited, and GeoIP-blocked traffic so the Grafana response breakdown graph is useful for screenshots.

I also added `run_k6.sh`, which:

- uses host `k6` if available
- otherwise falls back to the official Docker image
- writes a JSON summary file into `loadtest-artifacts/`

## Files changed

- `internal/observability/metrics.go`
- `cmd/engine/main.go`
- `cmd/analytics/main.go`
- `docker-compose.yml`
- `Makefile`
- `deployments/monitoring/prometheus/prometheus.yml`
- `deployments/monitoring/grafana/provisioning/datasources/prometheus.yml`
- `deployments/monitoring/grafana/provisioning/dashboards/dashboards.yml`
- `deployments/monitoring/grafana/dashboards/anti-fraud-overview.json`
- `scripts/loadtest/run_k6.sh`
- `scripts/loadtest/k6_real_click_ramp.js`
- `scripts/loadtest/k6_status_mix.js`
- `scripts/loadtest/README.md`
- `docs/MONITORING_LOADTEST.md`

## How to verify

Local monitoring:

```bash
COMPOSE_PROFILES=monitoring docker compose up --build -d
```

Open:

- `http://localhost:3000`
- `http://localhost:9091`

Run a local ramp:

```bash
bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

Run a mixed screenshot pass:

```bash
bash scripts/loadtest/run_k6.sh k6_status_mix.js
```
