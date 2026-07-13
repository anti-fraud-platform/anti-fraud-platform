# API Contract

## 1. Anti-Fraud Engine Core (Port 8080)

### `GET /health`
Healthcheck for Redis and PostgreSQL.

**Response (200 OK):**
```json
{
  "redis": "healthy",
  "postgres": "healthy",
  "geoip_loaded": true,
  "geoip_policy_enabled": true
}
```

### `GET /v1/challenge`
Returns a nonce and challenge_id for the JS Challenge (Tier 1). The frontend receives this before the click and solves it using client-side JS.

**Response (200 OK):**
```json
{
  "challenge_id": "abc123",
  "nonce": "random_string_here",
  "salt": "server_salt",
  "expires_at": 1720000000
}
```

### `POST /v1/click`
It receives click data and applies multi-level protection (Tier 1 + Tier 2).

**Request Body (JSON)**:
```json
{
  "ip": "1.2.3.4",
  "user_agent": "Mozilla/5.0...",
  "campaign_id": "camp_alpha_001",
  "timestamp": 1720000000,
  "challenge_id": "abc123",
  "challenge_token": "sha256_hash_here"
}
```

**Response (200 OK - click accepted):**
```json
{
  "status": "success",
  "message": "Click registered, routing to verification queue"
}
```

**Response (200 OK - click marked as suspicious, soft block):**
```json
{
  "status": "flagged",
  "message": "Click accepted for validation analysis pipeline"
}
```

**Response (403 Forbidden - hard block):**
```json
{
  "error": "Blocked by dynamic blacklist."
}
```
or
```json
{
  "error": "Blocked by GeoIP / ASN policy."
}
```

**Response (429 Too Many Requests - rate limit):**
```json
{
  "error": "Too many requests. Real-time anti-fraud trigger."
}
```

## 2. Analytics Service (Port 8081)

### `GET /health`
Healthcheck foor PostgreSQL.

**Response (200 OK):**
```json
{
  "postgres": "healthy"
}
```

### `GET /v1/analytics/stats`
General statistics, campaign aggregation, and top blocked IPs.

**Response (200 OK):**
```json
{
  "total_clicks": 12500,
  "allowed_count": 7520,
  "blocked_count": 4980,
  "blocked_bots": 4980,
  "saved_money_usd": 24900.00,
  "budget_saved": 24900.00,
  "previous_total_clicks": 10000,
  "previous_blocked_count": 3000,
  "total_clicks_delta_percent": 25.0,
  "blocked_count_delta_percent": 66.0,
  "js_challenge_blocked": 340,
  "header_heuristic_blocked": 88,
  "reason_breakdown": {
    "suspicious_agent": 12,
    "no_js_challenge": 340,
    "suspicious_headers": 88,
    "geoip_policy": 4,
    "rate_limit_exceeded": 900
  },
  "top_blocked_ips": [
    {
      "ip": "1.2.3.4",
      "blocked": 150,
      "total_requests": 200
    }
  ],
  "campaigns": [
    {
      "campaign_id": "camp_alpha_001",
      "total_clicks": 5000,
      "blocked_bots": 2000,
      "saved_money_usd": 10000.00
    }
  ]
}
```

### `GET /v1/analytics/logs`
Raw click logs with pagination and filters.

**Query params:**
- `page` (int, default 1)
- `limit` (int, default 20, max 100)
- `campaign_id` (string, optional)
- `is_bot` (bool, optional)
- `reason` (string, optional)
- `from` (RFC3339 or YYYY-MM-DD, optional)
- `to` (RFC3339 or YYYY-MM-DD, optional)

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": 1,
      "ip": "1.2.3.4",
      "campaign_id": "camp_alpha_001",
      "user_agent": "python-requests/2.31.0",
      "is_bot": true,
      "reason": "suspicious_agent",
      "processed_at": "2026-07-11T12:00:00Z"
    }
  ],
  "total": 12500,
  "page": 1,
  "limit": 20,
  "total_pages": 625
}
```

### `GET /v1/analytics/blacklist/summary`
Aggregate metrics for the Blacklist page.

**Response (200 OK):**
```json
{
  "total_blocked": 150,
  "geoip_policy_blocked": 4,
  "rate_limited": 900,
  "auto_blocked_24h": 12,
  "js_challenge_blocked": 340,
  "header_heuristic_blocked": 88
}
```

### `GET /v1/analytics/blacklist/ips`
List of blocked IPs (combination of `dynamic_blacklist` and `geoip_policy` blocks).

**Response (200 OK):**
```json
{
  "items": [
    {
      "ip": "1.2.3.4",
      "block_count": 150,
      "first_blocked": "2026-07-10 12:00",
      "last_blocked": "2026-07-11 14:30"
    }
  ],
  "total": 150
}
```

### `GET /v1/analytics/trend`
Aggregated daily traffic for the last 7 days, broken down by blocking reasons.

**Response (200 OK):**
```json
{
  "data": [
    {
      "date": "2026-07-05",
      "total_clicks": 1800,
      "allowed_count": 1200,
      "blocked_count": 600,
      "breakdown": {
        "suspicious_agent": 50,
        "rate_limit_exceeded": 500,
        "geoip_policy": 50
      }
    }
  ]
}
```

### `GET /v1/analytics/events`
The last 20 system events for the activity feed.

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "action_text": "User admin logged in",
    "created_at": "2026-07-11T12:00:00Z"
  }
]
```