# Railway Deployment Notes

This project can run on Railway, but the service layout is not exactly the same as local Docker Compose.

Local Compose gives us two things automatically:

1. bind-mounted files such as `deployments/blacklists/dirty_ips.txt`
2. Docker service DNS names like `engine` and `analytics`

Railway does not reuse that setup directly. Because of that, the safest Railway layout is:

- `engine`
- `analytics`
- `frontend`
- `postgres`
- `redis`

Optional:

- `nginx-engine` for the click simulator page

## What changed in the repo

- `Dockerfile.engine` now copies `dirty_ips.txt` into the image at `/app/data/dirty_ips.txt`
- `frontend` nginx config now uses runtime env variables instead of hardcoded upstream names
- `Dockerfile.nginx-engine` was added for a separate simulator service on Railway

## Recommended service layout

### 1. postgres

Create a Railway PostgreSQL service.

### 2. redis

Create a Railway Redis service.

### 3. engine

Connect the repo and set:

`RAILWAY_DOCKERFILE_PATH=Dockerfile.engine`

Service variables:

```env
PORT=8080
ENGINE_PORT=8080
BLACKLIST_PATH=/app/data/dirty_ips.txt
DB_HOST=${{Postgres.PGHOST}}
DB_PORT=${{Postgres.PGPORT}}
DB_USER=${{Postgres.PGUSER}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}
DB_NAME=${{Postgres.PGDATABASE}}
REDIS_HOST=${{Redis.REDISHOST}}
REDIS_PORT=${{Redis.REDISPORT}}
DB_BATCH_SIZE=1000
DB_BATCH_FLUSH_MS=500
DB_MAX_OPEN_CONNS=80
DB_MAX_IDLE_CONNS=20
REQUIRE_JS_CHALLENGE=true
REQUIRE_HEADER_CHECK=true
```

Keep this service private at first.

## 4. analytics

Create another service from the same repo and set:

`RAILWAY_DOCKERFILE_PATH=Dockerfile.analytics`

Service variables:

```env
PORT=8081
ANALYTICS_PORT=8081
DB_HOST=${{Postgres.PGHOST}}
DB_PORT=${{Postgres.PGPORT}}
DB_USER=${{Postgres.PGUSER}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}
DB_NAME=${{Postgres.PGDATABASE}}
REDIS_HOST=${{Redis.REDISHOST}}
REDIS_PORT=${{Redis.REDISPORT}}
DB_MAX_OPEN_CONNS=40
DB_MAX_IDLE_CONNS=10
```

This can stay private too.

## 5. frontend

Create another service from the same repo and set:

`RAILWAY_DOCKERFILE_PATH=frontend/Dockerfile`

Service variables:

```env
ANALYTICS_UPSTREAM=analytics.railway.internal:8081
ENGINE_UPSTREAM=engine.railway.internal:8080
```

Generate a public domain for this service. This is the main public entrypoint.

## 6. Optional simulator service

If you also want the standalone click simulator page, create one more service:

`RAILWAY_DOCKERFILE_PATH=Dockerfile.nginx-engine`

Service variables:

```env
ENGINE_UPSTREAM=engine.railway.internal:8080
```

Generate a public domain for it.

## GeoIP note

GeoIP databases are not bundled into the repo. That means the Railway deployment can run without GeoIP enrichment, but full GeoIP verification will only work after you provide the `.mmdb` files in a Railway-friendly way.
