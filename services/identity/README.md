# Identity service

Authentication and staff account management for SROOM, built with Go, Fiber v3, GORM, and PostgreSQL.

## Getting started

Requirements: the Go version specified in [go.mod](go.mod) and PostgreSQL. Run all commands from this service directory.

1. Create `.env` from [env.example](env.example) and set `DATABASE_URL`.
2. Export the same service `DATABASE_URL` in your shell, then run `migrate -path migrations -database "$DATABASE_URL" up`. The API does not run migrations automatically. For databases previously migrated manually, check the existing history first.
3. Start the API:

```sh
go run ./cmd/api
```

Configuration:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string, required |
| `APP_PORT` | HTTP port, defaults to `8081` |
| `APP_ENV` | Environment name, defaults to `development` |
| `OWNER_FULL_NAME`, `OWNER_EMAIL`, `OWNER_PASSWORD` | OWNER account details, used only by the seeder |

To create the OWNER, set the `OWNER_*` variables and run:

```sh
go run ./cmd/seed-owner
```

Running the seed again preserves the existing OWNER account.

## API

Local base URL: `http://localhost:8081/api/v1/identity`. Request bodies use JSON. Authenticated endpoints require the login token in the `Authorization: Bearer <token>` header.

| Method | Path | Access |
|---|---|---|
| POST | `/register` | Public |
| POST | `/login` | Public |
| GET | `/me` | Authenticated |
| POST | `/logout` | Authenticated |
| GET | `/admins` | OWNER |
| GET | `/admins/{id}` | OWNER |
| POST | `/admins` | OWNER |
| PATCH | `/admins/{id}` | OWNER |
| POST | `/change-password` | OWNER/ADMIN |
| POST | `/customer/change-password` | CUSTOMER |
| GET | `/customers` | OWNER/ADMIN |
| GET | `/customers/{id}` | OWNER/ADMIN |
| PATCH | `/customers/{id}/status` | OWNER/ADMIN |

- List ADMIN or CUSTOMER accounts: `search` matches name/email, `status=active|inactive`, `page` defaults to 1, and `limit` defaults to 20 (maximum 100). Results are ordered newest first.
- Create/update ADMIN accounts: `full_name`, `email`, `password`, `status`. Updates change only submitted fields; creation defaults to `active` status.
- Update a customer status with `{"status":"inactive"}` or `{"status":"active"}`.
- Change your own password: `current_password` and `new_password`.
- Successful responses use `data`; errors contain `error` and `request_id`. Successful logout/password changes return 204 without a body.

Payload and response details: [OpenAPI](../../packages/contracts/openapi/identity.v1.yaml).

## Account rules

- Public registration creates CUSTOMER accounts. A single OWNER is seeded; only OWNER can manage ADMIN accounts. The ADMIN role is fixed, and account deletion is not available through the API.
- Passwords require at least 8 characters and at most 72 UTF-8 bytes. Changing your own password requires the current password; OWNER does not need an ADMIN's current password to replace it.
- Sessions last 7 days. Inactive ADMIN and CUSTOMER accounts cannot log in again. Password/status changes preserve existing sessions until logout or expiry.

## Checks

```sh
go vet ./...
go test -race -count=1 ./...
go build ./...
```

Unit tests use fake repositories and GORM DryRun. Set `IDENTITY_TEST_DATABASE_URL` to a dedicated migrated test database with no OWNER to run the real API/seed PostgreSQL integration test. It creates its own accounts and removes only those accounts; never use the development database.
