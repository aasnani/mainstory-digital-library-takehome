# mainstory-digital-library-takehome

## Run (local)

```bash
go run .
```

Then hit:

- `GET /healthcheck` → `UP`

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