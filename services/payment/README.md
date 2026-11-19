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

RabbitMQ and Gateway routes will be added with the features that need them. The [HTTP contract](../../packages/contracts/openapi/payment.v1.yaml) documents the health endpoints.

For BNI VA setup and copyable Bruno requests, follow the [Midtrans sandbox guide](../../../docs/payment-sandbox-setup.md). `MIDTRANS_SERVER_KEY` is an optional placeholder for the upcoming integration; the bootstrap does not use it. Provider tests are pending local credentials.

## Migrations

Export the same `DATABASE_URL` in your terminal, then run from this directory:

```sh
migrate -path migrations -database "$DATABASE_URL" up
```

Tables are created in the `payment` schema. Each booking has one payment attempt; related tables store idempotency, provider observations, webhook receipts, and outbox events. Migrations are not run automatically at startup.

## Checks

```sh
go test ./...
go vet ./...
```

To include schema integration tests, use a **dedicated test database**, apply its migrations, and run:

```sh
migrate -path migrations -database "$PAYMENT_TEST_DATABASE_URL" up
go test -race -count=1 ./...
```

Export `PAYMENT_TEST_DATABASE_URL` before running these commands. Schema tests skip when it is unset; they do not fall back to `DATABASE_URL`.
