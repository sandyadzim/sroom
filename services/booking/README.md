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

Create a hold with `POST /api/v1/customer/bookings` and an `Idempotency-Key` UUID. Catalog must have an active car and active unit; no Catalog demo flag is needed. Each new hold reads current eligibility from Catalog. Later deactivation preserves existing bookings and in-flight holds already using an active snapshot. Successful retries return the saved booking. Use the verified payment integration below for Midtrans flows; demo payment/expiry mode remains a separate simulation.

## Verified payment confirmation

Apply migrations before enabling the consumer. Set `RABBITMQ_URL` to the same broker/vhost as Payment, keep `BOOKING_DEMO_PAYMENTS=false`, and use the same `PAYMENT_BOOKING_TOKEN` in both services. `PAYMENT_BASE_URL` defaults to `http://localhost:8084`. An empty broker URL disables the consumer; an active consumer rejects demo payment/expiry mode.

Booking consumes `payment.paid.v1`, verifies canonical facts through Payment's private API, and atomically commits inbox, confirmed/history, and a `booking.confirmed.v1` outbox event. Paid verified before the original deadline can confirm a pending booking with its allocation still blocking; late/unsafe cases go to review without changing allocations. Duplicate events do not repeat business effects.

Transient failures retry three times per delivery, then go to a durable DLQ before the original is acknowledged. Broker failure does not stop HTTP. Confirmed events are relayed from Booking outbox to Payment for invoice issuance; this paid consumer only confirms Booking. Inventory release follows the no-attempt expiry check below.

## Invoice integration

`GET /internal/v1/bookings/{id}/invoice-context` uses the existing pair token and returns the verified-confirmed rental snapshot, including payment references and confirmation event ID/time. Demo confirmations are unavailable. With `RABBITMQ_URL` set, a separate relay publishes only confirmed events every five seconds, up to 20 per batch; failure retains the original outbox event. Apply migration `000004_add_outbox_delivery` before startup. Payment owns invoice issuance/read APIs.

## Hold expiry

With `RABBITMQ_URL` configured and demo mode off, a separate startup/30-second worker asks Payment to close initiation for overdue holds. Durable no-attempt closure, or a matching authenticated BNI expiry lookup with saved release permission, allows Booking to atomically expire/release the allocation and save history/outbox. Uncertain dependencies and attempts without permission keep inventory blocked. The scanner rotates through batches of 50; release can lag the deadline. No new environment variable or Booking migration is needed.

Stop old Payment/Booking processes and apply Payment migration `000005_add_release_authorization` before restarting. No broker connection is needed for these HTTP checks. Paid learned after release permission goes to review without reclaiming inventory; demo expiry must stay disabled for Midtrans flows.

## Booking history

Use `GET /api/v1/customer/bookings` and `GET /api/v1/customer/bookings/{id}` through Gateway with a customer session. ADMIN and OWNER use `GET /api/v1/staff/bookings` and `GET /api/v1/staff/bookings/{id}`. Staff detail includes the customer ID, unit ID, and status history.

## Payment preparation

Set `PAYMENT_BOOKING_TOKEN` to enable `GET /internal/v1/bookings/{id}/payment-context` directly on Booking. The caller uses the same token as a Bearer credential. See the [private contract](../../packages/contracts/openapi/booking.internal.v1.yaml).

General provider reconciliation remains deferred; the expiry worker checks overdue attempts through Payment.

## Demo payments

Set `BOOKING_DEMO_PAYMENTS=true` and `PAYMENT_BOOKING_TOKEN`, then restart Booking. Call `POST /internal/v1/bookings/{id}/demo-confirmation` directly with that Bearer token and JSON `{"amount_idr":800000,"currency":"IDR"}` using the booking's actual total. The hold must be pending, before its deadline, and still allocated. Successful retries return the original confirmation time.

This simulates payment and records `booking.demo_confirmed.v1` in the outbox. Confirmed inventory stays blocked. The same mode runs expiry at startup and every 30 seconds, processing up to 50 pending holds per check. Due holds are assumed unpaid, become `expired`, release their units, and record `booking.demo_expired.v1`. Backlogs or database failures can delay release beyond the 15-minute deadline. Restart resumes from saved state.

Use a dedicated local demo database. The mode is disabled by default; saved records remain when it is disabled. The confirmation endpoint is unavailable through Gateway. This mode only simulates Booking state; use the verified payment integration above for Midtrans, RabbitMQ, invoice/email and provider-backed release.

## Staff reports

ADMIN/OWNER can read `/api/v1/staff/overview` and `/api/v1/staff/reports` through Gateway. Reports filter booking creation dates in WITA; default seven dates, maximum 366. Confirmed value/conversion exclude demo confirmations without canonical Payment references. Reads use saved Booking data and never call Midtrans.

## Checks

Run `go test ./...` from this directory.

To also test the allocation schema, export `BOOKING_TEST_DATABASE_URL` for a dedicated PostgreSQL test database:

```sh
migrate -path migrations -database "$BOOKING_TEST_DATABASE_URL" up
go test -count=1 ./...
```

Relay integration tests require a separate migrated `BOOKING_OUTBOX_TEST_DATABASE_URL` because relay polls all pending confirmed events. Do not reuse the ordinary test database.

When running all PostgreSQL suites against the same Booking test database, use `-p 1` to serialize packages: demo expiry and scanner tests otherwise see each other's fixtures. This does not change assertions or production worker concurrency.
