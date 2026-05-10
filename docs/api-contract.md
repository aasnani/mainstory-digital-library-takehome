# API contract (frontend / Lovable)

Single source of truth for browser clients. **Update this file in the same change series as any HTTP route, request/response shape, status code, auth, or CORS behavior change.**

## Overview

- **Audience**: SPAs (e.g. Lovable-generated UI) and integrators.
- **Base path**: `/api/v1` (all JSON APIs below are under this prefix unless noted).
- **Auth model**: **Mocked** — no real passwords. Register and login use **email only**. Production would replace this with real credentials and token refresh.

## Environment

| Variable (backend) | Required | Description |
|--------------------|----------|-------------|
| `DATABASE_URL` | yes | PostgreSQL connection URL (libpq / `postgresql://...`). |
| `JWT_SECRET` | yes | HMAC key for signing JWTs (use a long random string). |
| `PORT` | no | Listen port; default **8080**. |
| `JWT_EXPIRY_HOURS` | no | Access token lifetime in hours; default **24**. |
| `CORS_ALLOW_ORIGIN` | no | `Access-Control-Allow-Origin` value; default **`*`**. For production, set to your frontend origin (e.g. `https://app.example.com`). |

**Frontend**: configure a single **API base URL** (e.g. `VITE_API_BASE_URL` or your host’s env) and prefix paths below with it, e.g. `https://api.example.com/api/v1/auth/login`.

**Operators**: how to obtain connection strings and secrets, longer **`curl`** walkthroughs, and **`go test`** / Postgres integration tests are documented in **[README.md](../README.md)** (keep this file aligned when API behavior changes).

## Authentication

### Bearer JWT

Protected endpoints expect:

```http
Authorization: Bearer <access_token>
```

There must be exactly one space after `Bearer`.

### Obtaining tokens

Call **`POST /api/v1/auth/register`** or **`POST /api/v1/auth/login`** with body `{ "email": "user@example.com" }`.

Success response (**register**: HTTP **201**, **login**: HTTP **200**):

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "MEMBER"
  }
}
```

- **`expires_in`**: seconds until access token expiry (matches server `JWT_EXPIRY_HOURS`).
- **`user.role`**: `MEMBER`, `LIBRARIAN`, or `ADMIN` (register always creates **MEMBER**).

### JWT claims (reference)

Access tokens are HS256 JWTs. Claims relevant to clients:

| Claim | Meaning |
|-------|---------|
| `sub` | User id (UUID string). |
| `role` | `MEMBER`, `LIBRARIAN`, or `ADMIN`. |
| `exp` | Expiration time (Unix seconds). |
| `iat` | Issued-at (Unix seconds). |

You may decode the JWT in the browser **only for UI hints** (e.g. hide admin nav). **Authorization is always enforced by the API.**

### Token storage (browser)

The server does not store the token. Typical patterns:

- **`sessionStorage`**: cleared when the tab closes; good for demos.
- **`localStorage`**: survives refresh; clear on **401**.
- **Memory only**: most resilient to theft via XSS from persisted storage; lost on full reload.

On **401 Unauthorized**, clear the stored token and send the user through login/register again.

## CORS

If the UI origin differs from the API origin, the backend must send `Access-Control-Allow-Origin`. Configure **`CORS_ALLOW_ORIGIN`** to your frontend origin when not using `*`. This API uses **Bearer tokens only** (no cookie credentials required for MVP).

## Roles and admin

- **Register** creates **`MEMBER`** only.
- **ADMIN** (and role changes by admins) are enforced server-side.
- **There is no API that bootstraps admin.** Operators promote users via SQL (see README). After promotion, **`POST /auth/login`** returns a JWT whose **`role`** claim is **`ADMIN`**.

## Error envelope

Failed requests return JSON:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human-readable detail"
  }
}
```

## HTTP status reference

| Code | When |
|------|------|
| **400** | Invalid JSON, invalid UUID path param, invalid query (`limit`/`offset`), validation errors. |
| **401** | Missing/invalid `Authorization`, unknown email on login, expired JWT. |
| **403** | Authenticated but not allowed (e.g. non-admin listing users, self attempting role change). |
| **404** | User id not found. |
| **409** | Email already registered; email conflict on update; **cannot delete user** when **entitlements** rows reference that user. |
| **500** | Unexpected server error. |

## Endpoints

### Health (no version prefix)

| Method | Path | Auth | Success |
|--------|------|------|---------|
| GET | `/healthcheck` | none | **200** text body `UP` |

### Auth

| Method | Path | Auth | Body | Success |
|--------|------|------|------|---------|
| POST | `/api/v1/auth/register` | none | `{ "email": string }` | **201** + auth payload + user |
| POST | `/api/v1/auth/login` | none | `{ "email": string }` | **200** + auth payload + user |

### Users

| Method | Path | Auth | Body | Success |
|--------|------|------|------|---------|
| GET | `/api/v1/users/me` | Bearer | — | **200** `{ "id", "email", "role" }` |
| PATCH | `/api/v1/users/me` | Bearer | `{ "email"?: string, "role"?: string }` | **200** updated user. Non-admin: **`role` must not be sent** (forbidden). |
| GET | `/api/v1/users` | Bearer **ADMIN** | Query: `limit` (1–100, default 50), `offset` (≥0, default 0) | **200** `{ "users": [ ... ] }` |
| GET | `/api/v1/users/:id` | Bearer (**ADMIN** or **self**) | — | **200** user object |
| PATCH | `/api/v1/users/:id` | Bearer (**ADMIN** or **self**) | `{ "email"?: string, "role"?: string }` | **200** updated user. Self cannot escalate **role** without admin. |
| DELETE | `/api/v1/users/:id` | Bearer **ADMIN** | — | **204** no body. **409** if user still referenced by **entitlements**. |

**User JSON shape:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "MEMBER"
}
```

## Examples (`curl`)

Replace host, and export a real `DATABASE_URL` / `JWT_SECRET` before starting the server.

```bash
export DATABASE_URL='postgresql://...'
export JWT_SECRET='dev-secret-change-me-use-long-random-string'
go run .
```

Register and call **`/users/me`**:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"reader@example.com"}'

TOKEN='<paste access_token from response>'

curl -s http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

Login:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"reader@example.com"}'
```

Admin list (after promoting user to **ADMIN** in the database — see README):

```bash
curl -s 'http://localhost:8080/api/v1/users?limit=10&offset=0' \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Frontend checklist (Lovable)

1. Set **API base URL** from env.
2. After login/register, persist **`access_token`** and send **`Authorization: Bearer`** on all **`/api/v1`** calls except auth endpoints.
3. Handle **401** by clearing token and showing login/register.
4. Use **`GET /users/me`** for current user; do not trust user-typed UUIDs for identity.
5. Gate admin UI on **`role`** from **`/users/me`** or decoded JWT **only as UX**; rely on **403** from API if misconfigured.
