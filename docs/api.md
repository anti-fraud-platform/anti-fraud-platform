# API Contract

## 1. Anti-Fraud Engine Core (Port 8080)

### `POST /v1/click`
Receives click data and makes a decision: block or allow

**Request Body (JSON)**:
```json
{
  "user_id": "string",
  "ip": "string",
  "user_agent": "string",
  "timestamp": "2023-10-27T10:00:00Z"
}
```
**Response (200 OK):**

```json
{
  "status": "allowed"
}
```

## 2. Analytics Service (Port 8081)

### `GET /v1/analytics/stats`

**Response (200 OK):**
```json
{
  "total_clicks": 12500,
  "blocked_bots": 4980,
  "saved_money_usd": 24900.00
}
