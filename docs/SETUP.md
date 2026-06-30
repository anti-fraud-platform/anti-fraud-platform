# Local Setup

This guide is for a clean clone on a new machine. It assumes nothing except Git, Go, and Docker.

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

You do not need both Compose commands. One of them is enough.
On this machine, `docker-compose` exists while `docker compose` does not, so use the command that your setup actually provides.

## 2. Clone the repository

```bash
git clone git@github.com:kage-ops-dev/anti-fraud-platform.git
cd anti-fraud-platform
```

## 3. Build and start everything

Start the stack:

```bash
docker-compose up --build
```

If your Docker installation uses the Compose plugin form:

```bash
docker compose up --build
```

This should build and start:

- `antifraud-engine`
- `antifraud-analytics`
- `antifraud-frontend`
- `antifraud-postgres`
- `antifraud-redis`

## 4. Check that the services are up

List containers:

```bash
docker-compose ps
```

Expected state:

- `engine` is `Up`
- `analytics` is `Up`
- `frontend` is `Up`
- `postgres` is `Up (healthy)`
- `redis` is `Up (healthy)`

Open the frontend:

```text
http://localhost:3001
```

Check analytics:

```bash
curl http://localhost:8082/v1/analytics/stats
```

Check the click endpoint:

```bash
curl -X POST http://localhost:8080/v1/click \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":"demo_campaign","user_agent":"manual-check"}'
```

If everything is wired correctly, the endpoint should return a JSON success response.

## 5. Common local issues

### Compose command mismatch

If `docker compose` says `unknown command`, use:

```bash
docker-compose up --build
```

### Port conflicts

This project binds these host ports:

- `3001` -> frontend
- `8080` -> engine
- `8082` -> analytics
- `5433` -> postgres
- `6380` -> redis

If one of them is already taken, stop the conflicting process or change the host-side port in `docker-compose.yml`.

### Frontend dependency registry

The frontend Docker build uses the default npm registry.
That avoids one extra moving part on a fresh machine.
If package download fails inside the corporate network, check the corporate proxy and certificate chain first.
In this environment, Docker image pulls and package downloads can fail because of corporate proxy or TLS certificate policy rather than because of a repository bug.

### Docker daemon access

If Compose cannot talk to Docker on macOS, check whether Docker Desktop or Colima is running and whether your Docker context points to the right socket.

## 6. Clean rerun from scratch

If you want to repeat the setup with fresh containers but keep the existing Postgres volume:

```bash
docker-compose down
docker-compose up --build
```

If you want to test first-run initialization again, including database bootstrap, remove the named volume too:

```bash
docker-compose down -v
docker volume rm antifraud_pg_data
docker-compose up --build
```

Use the destructive version only when you intentionally want a clean database.
