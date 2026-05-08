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