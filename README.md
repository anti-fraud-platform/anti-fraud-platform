# Anti-Fraud Platform

[![CI](https://github.com/kage-ops-dev/anti-fraud-platform/actions/workflows/ci.yml/badge.svg)](https://github.com/kage-ops-dev/anti-fraud-platform/actions/workflows/ci.yml)

This repository contains a small anti-fraud demo stack for ad-click traffic.

- `engine` accepts click events on port `8080`
- `analytics` serves stats and logs from PostgreSQL
- `frontend` shows the dashboard through nginx
- `postgres` stores click logs
- `redis` stores rate-limit counters

## What you need

Install these tools first:

- Go `1.26.x`
- Docker Desktop, or Docker Engine with Compose support
- Git

On some machines the Compose command is `docker compose`. On others it is `docker-compose`.
Check which one you have:

```bash
docker compose version
docker-compose version
```

Use the one that works on your machine in the commands below.

## Clone and start the stack

Clone the repository and move into the project root:

```bash
git clone git@github.com:kage-ops-dev/anti-fraud-platform.git
cd anti-fraud-platform
```

Start the full local stack:

```bash
docker-compose up --build
```

If your machine uses the plugin form instead of the legacy binary, run:

```bash
docker compose up --build
```

When the stack is healthy, these endpoints should be reachable:

- frontend: `http://localhost:3001`
- engine: `http://localhost:8080/v1/click`
- analytics: `http://localhost:8082/v1/analytics/stats`
- postgres: `localhost:5433`
- redis: `localhost:6380`

For a clean-room setup check, follow the full step-by-step guide in [docs/SETUP.md](docs/SETUP.md).

## Run the Go test suite

Run all Go tests from the repository root:

```bash
go test ./...
```

Expected result:

```text
?   	anti-fraud/cmd/analytics	[no test files]
?   	anti-fraud/cmd/engine	[no test files]
?   	anti-fraud/cmd/generator	[no test files]
?   	anti-fraud/internal/bloom	[no test files]
ok  	anti-fraud/internal/engine	0.xxxs
?   	anti-fraud/internal/logger	[no test files]
?   	anti-fraud/internal/models	[no test files]
```

The exact test duration will differ, but `internal/engine` should report `ok` and the command should exit with code `0`.

## CI

GitHub Actions runs on every push to `main` and every pull request targeting `main`.
The workflow is in [.github/workflows/ci.yml](.github/workflows/ci.yml) and does two checks:

- `go build ./...`
- `go test ./...`

If either command fails, the workflow fails.

## Remote deployment status

The repository changes in this branch prepare the project for local verification and CI.
The public deployment required by Task 6 is still an external step because it needs access to a hosting account or VM.

For this stack, the most practical hosted options are:

- `Railway`: easiest path if you want managed Postgres and Redis, but the multi-service setup needs environment wiring and volume checks
- `Render`: workable, but usually slower to set up for a small multi-container stack
- `Fly.io`: flexible, but more manual because you need to think about process layout, volumes, and service networking

If the team does not get VM access in time, `Railway` is the lowest-friction fallback for a public demo in this project.
