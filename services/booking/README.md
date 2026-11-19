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

To create a demo hold, set `CATALOG_DEMO_HOLDS=true` in Catalog and restart it. Send `POST /api/v1/customer/bookings` with an `Idempotency-Key` UUID. Keep Catalog demo mode active while demo data exists. Booking's demo payment mode below controls automatic expiry; otherwise overdue holds stay blocking.

## Verified payment confirmation

Apply migrations before enabling the consumer. Set `RABBITMQ_URL` to the same broker/vhost as Payment, keep `BOOKING_DEMO_PAYMENTS=false`, and use the same `PAYMENT_BOOKING_TOKEN` in both services. `PAYMENT_BASE_URL` defaults to `http://localhost:8084`. An empty broker URL disables the consumer; an active consumer rejects demo payment/expiry mode.

Booking consumes `payment.paid.v1`, verifies canonical facts through Payment's private API, and atomically commits inbox, confirmed/history, and a `booking.confirmed.v1` outbox event. Paid verified before the original deadline can confirm a pending booking with its allocation still blocking; late/unsafe cases go to review without changing allocations. Duplicate events do not repeat business effects.

Transient failures retry three times per delivery, then go to a durable DLQ before the original is acknowledged. Broker failure does not stop HTTP. Confirmed events remain in Booking outbox until invoice delivery work; no invoice or inventory release occurs here. See [setup and Bruno](../../../docs/payment-confirm-booking-bruno.md). Real RabbitMQ/sandbox verification is pending.

## Booking history

Use `GET /api/v1/customer/bookings` and `GET /api/v1/customer/bookings/{id}` through Gateway with a customer session. ADMIN and OWNER use `GET /api/v1/staff/bookings` and `GET /api/v1/staff/bookings/{id}`. Staff detail includes the customer ID, unit ID, and status history.

## Payment preparation

Set `PAYMENT_BOOKING_TOKEN` to enable `GET /internal/v1/bookings/{id}/payment-context` directly on Booking. The caller uses the same token as a Bearer credential. See the [private contract](../../packages/contracts/openapi/booking.internal.v1.yaml).

Provider reconciliation follows the future Payment integration.

## Demo payments

Set `BOOKING_DEMO_PAYMENTS=true` and `PAYMENT_BOOKING_TOKEN`, then restart Booking. Call `POST /internal/v1/bookings/{id}/demo-confirmation` directly with that Bearer token and JSON `{"amount_idr":800000,"currency":"IDR"}` using the booking's actual total. The hold must be pending, before its deadline, and still allocated. Successful retries return the original confirmation time.

This simulates payment and records `booking.demo_confirmed.v1` in the outbox. Confirmed inventory stays blocked. The same mode runs expiry at startup and every 30 seconds, processing up to 50 pending holds per check. Due holds are assumed unpaid, become `expired`, release their units, and record `booking.demo_expired.v1`. Backlogs or database failures can delay release beyond the 15-minute deadline. Restart resumes from saved state.

Use a dedicated local demo database. The mode is disabled by default; saved records remain when it is disabled. The confirmation endpoint is unavailable through Gateway. Provider verification, RabbitMQ publication, invoices, email, and real-payment release policies are pending.

## Checks

Run `go test ./...` from this directory.

To also test the allocation schema, export `BOOKING_TEST_DATABASE_URL` for a dedicated PostgreSQL test database:

```sh
migrate -path migrations -database "$BOOKING_TEST_DATABASE_URL" up
go test -count=1 ./...
```
