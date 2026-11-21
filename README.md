# SROOM

SROOM is a self-drive car rental platform for Samarinda, Indonesia. Rentals are daily and self-drive only, displayed times use WITA (UTC+08:00), and payments use the Midtrans sandbox. The repository contains a customer and staff web application, a Gateway, and four Go services for identity, catalog, booking, and payment.

The customer flow covers browsing the fleet, choosing a schedule, creating a 15-minute hold, paying through a Midtrans sandbox BNI Virtual Account, and reading the confirmed booking and invoice. Staff manage the fleet, customers, bookings, payments, and operational reports.

## Architecture

```text
Browser
  │
  ▼
apps/web :5173  ──►  services/gateway :8080
                         ├──► services/identity :8081
                         ├──► services/catalog :8082
                         ├──► services/booking :8083
                         └──► services/payment :8084
```

| Component  | Port | Responsibility                                                     |
| ---------- | ---: | ------------------------------------------------------------------ |
| Web        | 5173 | SvelteKit customer and staff interface                             |
| Gateway    | 8080 | Browser entry point, session cookies, routing, bounded composition |
| Identity   | 8081 | Accounts, roles, sessions, password changes                        |
| Catalog    | 8082 | Brands, cars, units, banners, featured cars                        |
| Booking    | 8083 | Quotes, holds, allocations, booking lifecycle                      |
| Payment    | 8084 | BNI VA attempts, verified Midtrans outcomes, invoices              |
| PostgreSQL | 5432 | Service-owned databases and migrations                             |
| Redis      | 6379 | Optional Catalog read cache                                        |
| RabbitMQ   | 5672 | Durable payment/booking integration events                         |

PostgreSQL is the source of truth for allocations and payment state. Redis never owns money, bookings, or inventory. RabbitMQ delivery is at-least-once; consumers are idempotent.

## Services

### Web — `apps/web`

SvelteKit application for public catalog browsing, customer registration and account management, checkout, BNI VA payment, booking history, and invoices. Staff pages cover fleet management, customers, bookings, payments, invoices, Overview, and Reports. See the [web README](apps/web/README.md).

### Gateway — `services/gateway`

Single browser-facing HTTP entry point. It serves public, customer, and staff routes, converts HttpOnly session cookies to Bearer credentials, enforces allowed origins, forwards request IDs, applies body/time limits, and composes bounded Overview responses. It exposes `/api/v1/...` and `/webhooks/midtrans`; private service routes stay internal. See the [Gateway README](services/gateway/README.md).

### Identity — `services/identity`

Owns accounts, password hashes, roles, sessions, customer profiles, and staff account administration. Public signup creates CUSTOMER accounts; one OWNER is seeded and can manage ADMIN accounts. It provides login, logout, session verification, profile, customer status, and password-change operations. See the [Identity README](services/identity/README.md).

### Catalog — `services/catalog`

Owns brands, vehicle models, physical units, daily rates, photos, banners, and featured-car selections. It exposes public catalog reads and staff management APIs, stores images in Cloudflare R2, and provides a private Booking snapshot and staff fleet summary. Optional Redis caching applies to public brand reads. See the [Catalog README](services/catalog/README.md).

### Booking — `services/booking`

Owns rental quotes, 15-minute holds, physical-unit allocation, booking status, agreed schedules, terms acceptance, status history, and staff booking reports. PostgreSQL constraints prevent overlapping active allocations. Workers release eligible overdue holds; confirmed bookings remain allocated. See the [Booking README](services/booking/README.md).

### Payment — `services/payment`

Owns BNI Virtual Account attempts, idempotent initiation, verified Midtrans callbacks and manual refresh, provider observations, payment review facts, immutable invoices, and optional email delivery jobs. It never allocates units and never treats an unverified callback or redirect as settlement. See the [Payment README](services/payment/README.md).

## Main workflows

### Customer rental

1. Browse the public Catalog and choose a car.
2. Sign in or register, then open checkout for that car.
3. Request a backend quote for a WITA schedule; pickup starts tomorrow.
4. Accept rental terms and create a 15-minute hold.
5. Pay the BNI Virtual Account in the Midtrans sandbox.
6. Check payment status. A verified paid Payment may briefly precede Booking confirmation.
7. Read the confirmed Booking, booking detail, and issued invoice; print or save it as PDF.

### Staff operations

ADMIN and OWNER users manage fleet data, review customer accounts, inspect bookings and payments, read issued invoices, and view Overview and Reports. OWNER alone can manage ADMIN accounts. Transaction screens are read-only; they do not confirm payments, release units, refund, or change booking status.

### Payment confirmation

Payment records a verified provider outcome and publishes `payment.paid.v1` through its outbox. Booking consumes the event, verifies canonical facts, and confirms an eligible pending booking atomically. Booking then publishes `booking.confirmed.v1`; Payment consumes it, issues one immutable invoice, and may queue an optional email job. Duplicate delivery must not create duplicate invoices or state transitions.

## Technology

- **Backend:** Go 1.27.3, Gin for Gateway, Catalog, Booking, and Payment, Fiber v3 for Identity, GORM, PostgreSQL.
- **Frontend:** SvelteKit 3, Svelte 5, TypeScript, Tailwind CSS v4, Bun, TanStack Query, Axios.
- **Integration:** REST/JSON through Gateway, OpenAPI contracts, RabbitMQ events, Midtrans sandbox.
- **Configuration:** one `.env` per service and web app, with `.env.example`/`env.example` templates.

## Repository layout

```text
project/
├── apps/
│   └── web/                    # SvelteKit customer and staff application
├── services/
│   ├── gateway/                # Public HTTP entry point
│   ├── identity/               # Authentication and account management
│   ├── catalog/                # Vehicle catalog and fleet metadata
│   ├── booking/                # Availability, holds, and booking state
│   └── payment/                # Payment attempts and invoices
├── packages/
│   └── contracts/openapi/      # Public and internal OpenAPI contracts
├── go.work                     # Optional local Go workspace
├── go.work.sum
├── .editorconfig
├── .gitignore
└── README.md
```

Each service is its own Go module with its own `go.mod`, `go.sum`, migrations, tests, and README. The web app has its own `package.json` and lockfile.

## Requirements

- Go 1.27.3
- Bun
- PostgreSQL and the `migrate` CLI
- Redis when Catalog caching is enabled
- RabbitMQ for the verified payment confirmation and invoice flow
- A Midtrans sandbox account for BNI Virtual Account testing

SMTP, Mailpit, and a public webhook tunnel are optional for local frontend testing.

## Run locally

1. Create the PostgreSQL databases used by the services and apply each service's migrations. Database names and migration commands are documented in the service READMEs.
2. Create configuration files from the examples in each directory:

   - `services/identity/env.example`
   - `services/catalog/env.example`
   - `services/booking/.env.example`
   - `services/payment/.env.example`
   - `services/gateway/env.example`
   - `apps/web/env.example`

   Use the same `PAYMENT_BOOKING_TOKEN` in Booking and Payment, and the same `BOOKING_CATALOG_TOKEN` in Booking and Catalog.

3. Start Identity. To create the seeded OWNER, set the `OWNER_*` variables and run the seed command once.

   ```sh
   cd services/identity
   go run ./cmd/seed-owner
   go run ./cmd/api
   ```

4. Start the remaining services from their directories, each in its own terminal:

   ```sh
   cd services/catalog
   go run .
   ```

   ```sh
   cd services/booking
   go run .
   ```

   ```sh
   cd services/payment
   go run .
   ```

   ```sh
   cd services/gateway
   go run .
   ```

5. Start the web app.

   ```sh
   cd apps/web
   bun install
   bun run dev
   ```

6. Open `http://localhost:5173`.

For the full payment flow, set `BOOKING_DEMO_PAYMENTS=false`, run RabbitMQ, keep the Booking and Payment consumers active, and use the Midtrans sandbox BNI VA simulator. Gateway, Identity, and Catalog must also be running.

## Health checks

| Service | Endpoint                                                                          |
| ------- | --------------------------------------------------------------------------------- |
| Gateway | `GET http://localhost:8080/health`                                                |
| Catalog | `GET http://localhost:8082/health`                                                |
| Booking | `GET http://localhost:8083/health/live`, `GET http://localhost:8083/health/ready` |
| Payment | `GET http://localhost:8084/health/live`, `GET http://localhost:8084/health/ready` |

Booking and Payment readiness checks include PostgreSQL. Identity has no separate health route.

## Configuration notes

- Gateway `WEB_ORIGINS` must contain the exact web origin, for example `http://localhost:5173`.
- Web `PUBLIC_GATEWAY_BASE_URL` defaults to the current hostname on port `8080`.
- Customer and staff sessions use separate HttpOnly cookies under `/api/v1/customer` and `/api/v1/staff`.
- Keep `.env` files and provider credentials out of version control. Never expose the Midtrans server key or service tokens in browser code.
- A checked-in `go.work` is not required. It is a local convenience for working across service modules; CI and releases should build each module independently.

## Search indexing

This project is configured for a private deployment:

- `apps/web/static/robots.txt` disallows all crawlers.
- The global HTML shell includes a `noindex, nofollow, noarchive, nosnippet, noimageindex` robots meta tag.
- The SvelteKit server hook adds `X-Robots-Tag` to every response.
- Gateway responses also include `X-Robots-Tag`.

Do not add a sitemap or public indexing metadata. If a hostname was indexed before this configuration was deployed, Google may need a removal request or a temporary crawlable `noindex` state before indexing disappears.

## Contracts

HTTP contracts live in `packages/contracts/openapi/`:

- `gateway.v1.yaml` — browser-facing routes
- `identity.v1.yaml`, `catalog.v1.yaml`, `booking.v1.yaml`, `payment.v1.yaml` — service APIs
- `catalog.internal.v1.yaml`, `booking.internal.v1.yaml`, `payment.internal.v1.yaml` — private service integrations

The Gateway exposes `/api/v1/...` routes and keeps internal service routes private. Owning services independently verify roles and ownership.

## Checks

Frontend:

```sh
cd apps/web
bun run check
bun run test
bun run lint
bun run build
```

Backend:

```sh
cd services/<service>
go vet ./...
go test ./...
go build ./...
```

Database, RabbitMQ, Redis, and provider integration tests require dedicated test configuration. See each service README for the required environment variables and serial execution notes.

## Further reading

- [Web application](apps/web/README.md)
- [Gateway](services/gateway/README.md)
- [Identity](services/identity/README.md)
- [Catalog](services/catalog/README.md)
- [Booking](services/booking/README.md)
- [Payment](services/payment/README.md)
