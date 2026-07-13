# CI Smoke Checks

Human-friendly entry points:

- `make ci-compose-up` builds and boots the stack
- `make ci-compose-smoke` runs the full smoke suite
- `make ci-compose-down` tears everything down

CI uses `docker-compose.ci.yml`, not the main `docker-compose.yml`.
Reason: GitLab runs Docker inside Docker, and nested bind mounts from `/builds/...`
are unreliable there. The CI compose file packages nginx assets and GeoIP data into
images instead of mounting them from the repo checkout.

If you want only one check, run one of these:

- `make ci-check-wait`
- `make ci-check-frontend`
- `make ci-check-simulator`
- `make ci-check-analytics`
- `make ci-check-challenge`
- `make ci-check-nginx-reresolve`
- `make ci-check-frontend-reresolve`

Low-level entry point: `scripts/ci/compose_smoke.sh`

Layout:

- `checks/01_wait_for_stack.sh` waits for frontend, analytics, and nginx to answer.
- `checks/02_frontend_shell.sh` verifies the React shell is served on `:3001`.
- `checks/03_simulator_page.sh` verifies the click simulator page is served on `:9090`.
- `checks/04_analytics_contract.sh` checks that analytics returns the required response fields.
- `checks/05_challenge_flow.sh` checks `/v1/challenge` and proves an unsolved click is flagged.
- `checks/06_nginx_reresolve.sh` recreates only the `engine` container and checks that nginx still reaches it.
- `checks/07_frontend_engine_proxy_reresolve.sh` checks that the frontend nginx proxy still reaches the recreated engine too.
- `lib/common.sh` holds the shared curl helpers and JSON assertions.

Each file proves one thing. If CI fails, you can usually start with the matching check file instead of reading one long shell script.
