# Payment Service

Go + Gin service for BNI Virtual Account initiation and verified Midtrans sandbox payment outcomes. Booking confirmation follows separately.

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

The [HTTP contract](../../packages/contracts/openapi/payment.v1.yaml) documents health, VA initiation, and webhooks. RabbitMQ is optional for asynchronous outbox delivery.

## Create a BNI VA

Set `MIDTRANS_SERVER_KEY` to your sandbox key and use the same `PAYMENT_BOOKING_TOKEN` in Payment and Booking. `IDENTITY_BASE_URL` defaults to `http://localhost:8081`; `BOOKING_BASE_URL` defaults to `http://localhost:8083`. Health works without integration credentials.

Start Gateway and send `POST /api/v1/customer/bookings/{id}/payments/va` with the customer cookie, an allowed Origin, a UUID `Idempotency-Key`, and JSON `{}`. Payment verifies the session and ownership, then uses Booking's amount and original deadline. The provider is fixed to the Midtrans sandbox.

The first ready VA returns `201`; existing attempts return `200`; uncertain initialization or review returns `202` without usable instructions. Retry the same booking/key. An uncertain attempt never creates a replacement charge; recovery follows later reconciliation work. Disable Booking's `BOOKING_DEMO_PAYMENTS` mode before trying Midtrans-backed holds.

See the [account setup guide](../../../docs/payment-sandbox-setup.md) and [Bruno flow](../../../docs/payment-create-va-bruno.md). Direct sandbox verification remains pending; automated tests use fake providers.

## Read payments

Customer: `GET /api/v1/customer/bookings/{id}/payment` through Gateway with the customer cookie. Returns stored payment state, or `not_started` for an owned booking without an attempt. VA instructions follow the same pending/deadline/review rules as create VA.

Staff: `GET /api/v1/staff/payments` and `GET /api/v1/staff/payments/{id}` with the ADMIN/OWNER cookie. Lists support `search`, `status`, `reconciliation=required|clear`, and `page`/`limit` (default 1/20, max 100), newest first. Detail includes safe observations and current Booking status; staff responses omit VA numbers.

Payment independently verifies Identity sessions. Customer reads and staff detail require Booking's private API; failure returns `503`. The list reads Payment only. GET never calls Midtrans or changes state, and needs no Midtrans key. See [Bruno read examples](../../../docs/payment-read-bruno.md).

## Check payment status

`POST /api/v1/customer/bookings/{id}/payment/refresh` through Gateway uses the customer cookie, an allowed Origin, and JSON `{}`. No idempotency key is needed. Payment verifies ownership, checks the existing sandbox order once, and applies the same verification/transaction rules as the webhook. The response matches customer GET.

No attempt returns `not_started`; already-paid attempts return stored facts without a lookup. Other attempts require the existing sandbox key. Temporary lookup failures return `503` and preserve state; committed mismatches return `200` with review required. Refresh never creates a VA, extends the deadline, or confirms Booking. Missing VA instructions are not recovered. No automatic worker or new configuration/migration is added. See [Bruno refresh examples](../../../docs/payment-refresh-bruno.md).

## Payment notifications

Gateway forwards `POST /webhooks/midtrans` to Payment. This provider callback uses signature verification and an authenticated Midtrans status lookup, without customer cookies or Origin checks. The body limit is 16 KiB; unknown provider fields are allowed.

Payment stores receipt hashes, sanitized observations, and verified state. Duplicate callbacks produce one `payment.paid.v1` outbox event per attempt; paid never regresses. Temporary failures return `503` for retry; committed review/unmatched receipts return `200`. No raw payload or signature is stored. When the relay is enabled, events are published to RabbitMQ; this endpoint does not wait for the broker, confirm Booking, or release inventory.

See [sandbox notification setup and checks](../../../docs/payment-sandbox-setup.md#8-notification-url). Direct sandbox verification remains pending; automated tests use fake providers. No additional environment variable or migration is needed for this callback.

## Event delivery

Optionally set `RABBITMQ_URL` to enable one outbox relay inside Payment. It sends up to 20 pending paid events every five seconds, one at a time, using persistent messages, mandatory routing, and publisher confirms. Failed/ambiguous delivery stays pending; retries retain the event ID. Invalid events are retained with `INVALID_EVENT` for inspection.

The durable `booking.payment-events.v1` queue receives `payment.paid.v1` through `sroom.domain.v1`. There is no Booking consumer yet. Broker outages do not stop HTTP; readiness remains PostgreSQL-only. This relay sends saved events and does not check Midtrans. No migration is needed. See [local Compose](../../infra/compose/README.md) and [backlog/recovery guide](../../../docs/payment-outbox-setup.md).

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

Relay persistence tests additionally require `PAYMENT_OUTBOX_TEST_DATABASE_URL`, a separate migrated test database because relay queries poll all pending events. RabbitMQ transport tests are opt-in with `PAYMENT_RABBITMQ_TEST_URL` pointing to a dedicated `sroom-test` vhost. Without these variables, the corresponding tests skip. Real RabbitMQ and Compose execution remain unverified until Docker is available.
