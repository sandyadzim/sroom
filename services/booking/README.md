# Booking Service

Go + Gin service for rental availability and reservations. The bootstrap includes HTTP liveness and PostgreSQL readiness checks.

## Run locally

From this directory, copy `.env.example` to `.env` and set `DATABASE_URL` to your local Booking PostgreSQL connection string. `APP_PORT` defaults to `8083`.

Export the same `DATABASE_URL` in your shell and apply the `booking` schema migrations with [golang-migrate](https://github.com/golang-migrate/migrate). The migration role needs permission to create the schema and install `btree_gist` if the extension is absent.

```sh
migrate -path migrations -database "$DATABASE_URL" up
```

Start the service:

```sh
go run .
```

In another terminal, with the default port:

```sh
curl -i http://localhost:8083/health/live
curl -i http://localhost:8083/health/ready
```

`/health/live` checks the HTTP process. `/health/ready` checks PostgreSQL and returns `503` while the database is unavailable.

## Checkout quote

Start Identity, Catalog, and Gateway. Booking uses `IDENTITY_BASE_URL` (default `http://localhost:8081`) to verify customer sessions. Use the same `BOOKING_CATALOG_TOKEN` in Booking and Catalog, and set `CATALOG_BASE_URL` if Catalog runs away from `http://localhost:8082`. Restart both services after changing the token.

Sign in through Gateway and send `POST /api/v1/customer/checkout/quotes` with the customer cookie and an allowed origin. See the [Booking request and response schemas](../../packages/contracts/openapi/booking.v1.yaml) and the [Gateway contract](../../packages/contracts/openapi/gateway.v1.yaml).

To create a demo hold, set `CATALOG_DEMO_HOLDS=true` in Catalog and restart it. Send `POST /api/v1/customer/bookings` with an `Idempotency-Key` UUID. Keep demo mode active while demo holds exist. Holds remain blocking until an expiry workflow can safely release them.

## Booking history

Use `GET /api/v1/customer/bookings` and `GET /api/v1/customer/bookings/{id}` through Gateway with a customer session. ADMIN and OWNER use `GET /api/v1/staff/bookings` and `GET /api/v1/staff/bookings/{id}`. Staff detail includes the customer ID, unit ID, and status history.

## Payment preparation

Set `PAYMENT_BOOKING_TOKEN` to enable `GET /internal/v1/bookings/{id}/payment-context` directly on Booking. The caller uses the same token as a Bearer credential. See the [private contract](../../packages/contracts/openapi/booking.internal.v1.yaml).

The expiry candidate query reads up to 50 overdue pending holds. Status changes and allocation release follow the Payment integration.

## Demo confirmation

Set `BOOKING_DEMO_PAYMENTS=true` and `PAYMENT_BOOKING_TOKEN`, then restart Booking. Call `POST /internal/v1/bookings/{id}/demo-confirmation` directly with that Bearer token and JSON `{"amount_idr":800000,"currency":"IDR"}` using the booking's actual total. The hold must be pending, before its deadline, and still allocated. Successful retries return the original confirmation time.

This simulates payment and records `booking.demo_confirmed.v1` in the outbox. Inventory stays blocked. Provider verification, RabbitMQ publication, invoices, email, and automatic expiry are pending. The endpoint is disabled by default and unavailable through Gateway. Use a dedicated local demo database; demo-confirmed records remain after disabling the endpoint.

## Checks

Run `go test ./...` from this directory.

To also test the allocation schema, export `BOOKING_TEST_DATABASE_URL` for a dedicated PostgreSQL test database:

```sh
migrate -path migrations -database "$BOOKING_TEST_DATABASE_URL" up
go test -count=1 ./...
```
