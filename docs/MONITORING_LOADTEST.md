# Monitoring and Load Test

## 1. Start monitoring locally

```bash
COMPOSE_PROFILES=monitoring docker compose up --build -d
```

What this starts in addition to the core stack:

- Prometheus on `http://localhost:9091`
- Grafana on `http://localhost:3000`
- node-exporter inside the monitoring profile

Default Grafana login:

- user: `admin`
- password: `admin`

The dashboard is provisioned automatically as:

- `Anti-Fraud / Anti-Fraud Platform Overview`

## 2. What the dashboard shows

- click request rate on `/v1/click`
- `200 / 403 / 429` response breakdown over time
- `p95` click latency
- engine goroutine count
- node CPU / memory usage
- Redis / PostgreSQL health as seen by the Go services

The Redis / PostgreSQL health panels come from the applications themselves, not from Docker healthchecks. That means the gauge answers the question: "can the running service still talk to this dependency right now?"

## 3. Start monitoring on the university VM

If you already have an SSH alias in `~/.ssh/config`:

```bash
ssh antifraud-vm
cd ~/apps/anti-fraud-platform
COMPOSE_PROFILES=monitoring docker compose up --build -d
```

If you want Grafana in the local browser through an SSH tunnel:

```bash
ssh -L 3000:localhost:3000 -L 9091:localhost:9091 antifraud-vm
```

Then open:

- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9091`

## 4. Run the real-click ramp test against the VM

From your laptop or from another machine that can reach the VM:

```bash
BASE_URL=http://10.93.26.161:9090 \
bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

Recommended first ramp:

```bash
BASE_URL=http://10.93.26.161:9090 \
STAGES=1m:10,2m:25,2m:50,2m:75,1m:100,1m:0 \
bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

What to extract for the report:

- the highest steady req/s before `429` starts rising sharply
- or the point where `p95` latency stops staying near its previous baseline

That number is your "max req/s before degradation" for the report.

## 5. Generate a clean `200 / 403 / 429` screenshot

After the main ramp, run a short mixed pass:

```bash
BASE_URL=http://10.93.26.161:9090 \
DURATION=90s \
bash scripts/loadtest/run_k6.sh k6_status_mix.js
```

This intentionally creates:

- `200` from solved real clicks
- `429` from rate-limited traffic
- `403` from a GeoIP-blocked IP through `X-Forwarded-For`

Use this run when you need a clear Grafana screenshot for the report.
