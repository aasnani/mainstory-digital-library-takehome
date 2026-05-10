# mainstory-digital-library-takehome

## Frontend / Lovable

All browser-facing API behavior (paths, JSON, auth, errors, CORS) is documented in **[docs/api-contract.md](docs/api-contract.md)**. Keep that file updated whenever the HTTP API changes.

## Configuration (backend)

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | **yes** | PostgreSQL URL (`postgresql://user:pass@host:port/dbname?sslmode=...`). |
| `JWT_SECRET` | **yes** | Secret for signing JWT access tokens (long random string). |
| `PORT` | no | Default **8080**. |
| `JWT_EXPIRY_HOURS` | no | Default **24**. |
| `CORS_ALLOW_ORIGIN` | no | Default **`*`**. Set to your frontend origin in production. |

## Run (local)

```bash
export DATABASE_URL='postgresql://...'
export JWT_SECRET='your-long-random-secret'
go run .
```

- `GET /healthcheck` → `UP`
- JSON API: **`/api/v1/...`** — see [docs/api-contract.md](docs/api-contract.md).

### Quick `curl`

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com"}'
# Use access_token from the response:
curl -s http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

## Admin user (manual, no API bootstrap)

Create or promote an admin with **`psql`** (or any SQL client). Example: set **`ADMIN`** for an existing login email:

```sql
UPDATE users SET role = 'ADMIN' WHERE lower(email) = lower('you@example.com');
```

Then **`POST /api/v1/auth/login`** with that email returns a JWT whose **`role`** claim is **`ADMIN`**.

## Build + run (deployment)

```bash
go build -tags netgo -ldflags '-s -w' -o app
./app
```

## Database migrations (Flyway)

Versioned SQL lives under `db/migration/` (for example `V1__initial_schema.sql`). Apply with the Flyway CLI pointed at your Postgres URL; see the Flyway docs for `-locations` and JDBC URLs.

Example (remote Postgres — replace host, database, user, and use a secure password source):

```bash
flyway \
  -locations=filesystem:db/migration \
  -url=jdbc:postgresql://HOST:5432/DATABASE \
  -user=USER \
  -password=PASSWORD \
  migrate
```
