# Booking Service

Go + Gin service for rental availability and reservations. The bootstrap includes HTTP liveness and PostgreSQL readiness checks.

## Run locally

From this directory, copy `.env.example` to `.env` and set `DATABASE_URL` to your local Booking PostgreSQL connection string. `APP_PORT` defaults to `8083`.

```sh
go run .
```

In another terminal, with the default port:

```sh
curl -i http://localhost:8083/health/live
curl -i http://localhost:8083/health/ready
```

`/health/live` checks the HTTP process. `/health/ready` checks PostgreSQL and returns `503` while the database is unavailable.

Run `go test ./...` from this directory.
