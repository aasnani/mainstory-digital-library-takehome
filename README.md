# mainstory-digital-library-takehome

## Frontend / Lovable

All browser-facing API behavior (paths, JSON, auth, errors, CORS) is documented in **[docs/api-contract.md](docs/api-contract.md)**. Keep that file updated whenever the HTTP API changes.

## Environment variables (reference)

The process reads **only** the standard environment (the app does **not** auto-load a `.env` file). Set variables in the shell, your process manager, or a tool like `direnv`.

### Must be set (server will not start without these)

| Variable | How to obtain / what to put |
|----------|------------------------------|
| **`DATABASE_URL`** | Full PostgreSQL connection string. **Local example:** `postgresql://USER:PASSWORD@127.0.0.1:5432/DBNAME?sslmode=disable`. **Managed (e.g. Render):** copy the “Internal” or “External” URL from the dashboard; use `sslmode=require` when the provider requires TLS. |
| **`JWT_SECRET`** | A long, random secret used to **sign** JWTs (not stored in the DB). Generate one e.g. `openssl rand -base64 48` and set it in production secrets—never commit it. |

### Optional (defaults shown)

| Variable | Default | Purpose |
|----------|---------|---------|
| **`PORT`** | `8080` | HTTP listen port. |
| **`JWT_EXPIRY_HOURS`** | `24` | Access-token lifetime in hours (`expires_in` in auth responses derives from this). |
| **`CORS_ALLOW_ORIGIN`** | `*` | Value sent as `Access-Control-Allow-Origin`. For production browser apps, set to your frontend origin (e.g. `https://app.example.com`). |

## Run (local)

```bash
export DATABASE_URL='postgresql://...'
export JWT_SECRET='your-long-random-secret'
go run .
```

- `GET /healthcheck` → `UP`
- JSON API: **`/api/v1/...`** — see [docs/api-contract.md](docs/api-contract.md).

## Manual API testing (`curl`)

Prerequisites: **Flyway migrations applied** to the DB in `DATABASE_URL`, server running as above.

### 1) Health

```bash
curl -s http://localhost:8080/healthcheck
```

### 2) Register → token → current user

```bash
BASE=http://localhost:8080

REGISTER_JSON=$(curl -s -X POST "$BASE/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo.reader@example.com","password":"securepass123"}')
echo "$REGISTER_JSON"

TOKEN=$(echo "$REGISTER_JSON" | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')

curl -s "$BASE/api/v1/users/me" \
  -H "Authorization: Bearer $TOKEN"
```

If you don’t have Python, copy **`access_token`** from the JSON manually into **`TOKEN`**.

### 3) Login (existing email)

```bash
curl -s -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo.reader@example.com","password":"securepass123"}'
```

### 4) Admin list (`ADMIN` role required)

Promote your user in SQL first (see below), then login again and use that JWT:

```bash
curl -s "$BASE/api/v1/users?limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 5) Negative checks

Wrong/missing token:

```bash
curl -s -o /dev/stderr -w "%{http_code}" "$BASE/api/v1/users/me"
curl -s "$BASE/api/v1/users/me" -H 'Authorization: Bearer invalid'
```

Expect **401** for missing/invalid Bearer token.

## Automated tests (Go)

All tests are **in-process** (no Docker, no live Postgres). CI runs `go test ./...` the same way you can locally.

```bash
go test ./...
```

## Admin user (manual, no API bootstrap)

Create or promote an admin with **`psql`** (or any SQL client). Example: set **`ADMIN`** for an existing login email:

```sql
UPDATE users SET role = 'ADMIN' WHERE lower(email) = lower('you@example.com');
```

Then **`POST /api/v1/auth/login`** with that email and password returns a JWT whose **`role`** claim is **`ADMIN`**.

## Build + run (deployment)

```bash
go build -tags netgo -ldflags '-s -w' -o app
./app
```

## Database migrations (Flyway)

Versioned SQL lives under `db/migration/` (for example `V1__initial_schema.sql`, `V2__users_password_hash.sql`). Apply with the Flyway CLI pointed at your Postgres URL; see the Flyway docs for `-locations` and JDBC URLs.

**`V2__users_password_hash.sql`** adds **`password_hash`** to **`users`**. If an older dev database still has email-only users without hashes, clear or migrate those rows before applying **`V2`** (the migration expects every user row to gain a password hash).

Example (remote Postgres — replace host, database, user, and use a secure password source):

```bash
flyway \
  -locations=filesystem:db/migration \
  -url=jdbc:postgresql://HOST:5432/DATABASE \
  -user=USER \
  -password=PASSWORD \
  migrate
```
