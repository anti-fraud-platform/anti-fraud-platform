# API Contract

## 1. Anti-Fraud Engine Core (Port 8080)

### `POST /v1/click`
Принимает данные клика, принимает решение: блокировать или пропустить.

**Request Body (JSON)**:
```json
{
  "campaign_id": "string",
  "user_agent": "string"
}
```
**Response (200 OK):**
```json
{
  "allowed": true,
  "reason": "passed_initial_check"
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
```
