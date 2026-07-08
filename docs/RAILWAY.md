# Railway Deployment Notes

This project can run on Railway, but the service layout is not exactly the same as local Docker Compose.

Local Compose gives us two things automatically:

1. Docker service DNS names like `engine` and `analytics`
2. local bridge networking between the services

Railway does not reuse that setup directly. Because of that, the safest Railway layout is:

- `engine`
- `analytics`
- `frontend`
- `postgres`
- `redis`

Optional:

- `nginx-engine` for the click simulator page

## What changed in the repo

- `Dockerfile.engine` now copies the MaxMind GeoIP databases into `/usr/share/GeoIP/`
- `frontend` nginx config now uses runtime env variables instead of hardcoded upstream names
- `Dockerfile.nginx-engine` was added for a separate simulator service on Railway
- both public nginx-based services now detect the DNS resolver from `/etc/resolv.conf` at container startup and re-resolve upstream service names instead of pinning one old IP forever

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
DB_HOST=${{Postgres.PGHOST}}
DB_PORT=${{Postgres.PGPORT}}
DB_USER=${{Postgres.PGUSER}}
DB_PASSWORD=${{Postgres.PGPASSWORD}}
DB_NAME=${{Postgres.PGDATABASE}}
REDIS_HOST=${{Redis.REDISHOST}}
REDIS_PORT=${{Redis.REDISPORT}}
REDIS_USER=${{Redis.REDISUSER}}
REDIS_PASSWORD=${{Redis.REDISPASSWORD}}
GEOIP_COUNTRY_DB_PATH=/usr/share/GeoIP/GeoLite2-Country.mmdb
GEOIP_CITY_DB_PATH=/usr/share/GeoIP/GeoLite2-City.mmdb
GEOIP_ASN_DB_PATH=/usr/share/GeoIP/GeoLite2-ASN.mmdb
GEOIP_BLOCKED_ASN_KEYWORDS=digitalocean,cloudflare,hetzner,ovh,linode,vultr,choopa,contabo,scaleway,oracle,amazon technologies,microsoft azure,google cloud
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

`UPSTREAM_RESOLVER` is optional. If you do not set it, the container auto-detects the nameserver from `/etc/resolv.conf`. Raw IPv6 resolvers are normalized to nginx-safe bracketed form automatically.

Generate a public domain for this service. This is the main public entrypoint.
Railway injects `PORT` automatically for public services. Do not hardcode it to `80`.

## 6. Optional simulator service

If you also want the standalone click simulator page, create one more service:

`RAILWAY_DOCKERFILE_PATH=Dockerfile.nginx-engine`

Service variables:

```env
ENGINE_UPSTREAM=engine.railway.internal:8080
```

`UPSTREAM_RESOLVER` is optional here too. The container auto-detects it on startup and normalizes raw IPv6 values automatically.

Generate a public domain for it.

## GeoIP note

The GeoIP databases are bundled in the repo and copied into the engine image during the Docker build. That means Railway gets the same GeoIP / ASN policy data as local Docker without an extra volume or manual upload step.
