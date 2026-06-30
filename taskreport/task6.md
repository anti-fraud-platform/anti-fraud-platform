# Task 6 Report

## What was deployed

The stack was deployed to the university VM `afplatform` running Ubuntu 22.04.
The deployment target is inside the Innopolis University network, so the services are reachable from the internal network and through the university VPN.

The deployed stack includes:

- `frontend` on port `3001`
- `analytics` on port `8082`
- `nginx_engine` click simulator on port `9090`
- `engine` as an internal-only service behind nginx
- `postgres` on port `5433`
- `redis` on port `6380`

## Live access details

Current VM address:

- `http://10.93.26.161:3001` - React dashboard
- `http://10.93.26.161:8082/v1/analytics/stats` - analytics API
- `http://10.93.26.161:9090` - engine simulator page

The engine is intentionally not exposed directly. Click traffic goes through the nginx simulator on port `9090`.

## What was verified

The following checks were executed directly on the VM:

```bash
docker ps
curl http://localhost:3001
curl http://localhost:8082/v1/analytics/stats
curl -X POST http://localhost:9090/click -H "Content-Type: application/json" -d '{}'
curl -X POST http://localhost:9090/bot/click -H "Content-Type: application/json" -d '{}'
```

Observed result:

- all containers were `Up`
- PostgreSQL and Redis were `healthy`
- the frontend served the built React app
- the analytics endpoint returned live JSON
- manual clicks were accepted
- bot clicks were flagged
- analytics counters changed after requests, confirming end-to-end flow through engine, Redis, batch logging, PostgreSQL, and analytics

## CI/CD wiring

GitLab shared runners were able to reach the VM over SSH from the university network.
An SSH smoke job succeeded with:

- VM host reachability
- key-based authentication
- access to the deployment directory

The next deploy step uses the same SSH path to:

```bash
git pull --ff-only origin main
docker-compose up --build -d
docker-compose ps
```

## Notes

This deployment is not intended for public internet access.
It is a stage environment for the course demo and is reachable from the university network or VPN, which matches the VM access constraints in the infrastructure request.
