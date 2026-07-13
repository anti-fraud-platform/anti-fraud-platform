# Load Testing

Two entry points:

- `k6_real_click_ramp.js`
  Uses the real `/v1/challenge` -> `/click` flow and ramps load until latency or error rate starts to climb.
- `k6_status_mix.js`
  Generates a mixed stream of `200`, `429`, and `403` responses so the Grafana breakdown panel is easy to read live.

Human-friendly wrapper:

```bash
bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js
```

```bash
bash scripts/loadtest/run_k6.sh k6_status_mix.js
```

If `k6` is installed on the host, the wrapper uses it directly.
Otherwise it falls back to the official `grafana/k6` container and saves the summary JSON into `loadtest-artifacts/`.

Useful environment variables:

- `BASE_URL=http://10.93.26.161:9090`
- `STAGES=1m:10,2m:25,2m:50,2m:75,1m:0`
- `ALLOWED_IPS=8.8.8.8,8.8.4.4,9.9.9.9,208.67.222.222`
- `RATE_LIMIT_IP=8.8.8.8`
- `GEO_BLOCKED_IP=1.1.1.1`
- `SOLVE_DELAY_MS=250`

Recommended Week 6 flow on the university VM:

1. Start the stack with monitoring enabled.
2. Open Grafana and keep the `Anti-Fraud Platform Overview` dashboard visible.
3. Run `k6_real_click_ramp.js` and note the req/s level where `429` begins to climb or `p95` latency stops staying flat.
4. Run `k6_status_mix.js` for a short pass so the `200/403/429` chart has all three live lines for the report screenshot.
