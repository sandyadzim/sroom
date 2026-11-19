# Payment Service

Go + Gin service with HTTP liveness and PostgreSQL readiness checks. Payment processing is not implemented yet.

## Run locally

From this directory, copy `.env.example` to `.env` and set `DATABASE_URL` to your local `rent-car-payment` PostgreSQL connection string. Existing environment variables take precedence over `.env`. `APP_PORT` defaults to `8084`.

```sh
go run .
```

In another terminal:

```sh
curl -i http://localhost:8084/health/live
curl -i http://localhost:8084/health/ready
```

Liveness returns `200` while HTTP is running. Readiness pings PostgreSQL with a two-second timeout, returning `200` when ready or `503 DATABASE_UNAVAILABLE` when unavailable. It recovers without restarting Payment. Invalid configuration stops startup with a safe error message.

Migrations are not needed yet. Midtrans keys, RabbitMQ, and Gateway routes will be added with the features that need them. The [HTTP contract](../../packages/contracts/openapi/payment.v1.yaml) documents the health endpoints.

## Checks

```sh
go test ./...
go vet ./...
```
