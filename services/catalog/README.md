# Catalog service

Vehicle catalog, brand management, and homepage banners and featured cars for SROOM, built with Go, Gin, GORM, PostgreSQL, and Cloudflare R2.

## Getting started

Use the Go version specified in [go.mod](go.mod), a Catalog PostgreSQL database, and the `migrate` CLI. Run these commands from this service directory.

1. Create `.env` from [env.example](env.example) and set `DATABASE_URL`. Make the same value available in your shell for migrations.
2. Apply migrations and start the API.

   ```sh
   migrate -path migrations -database "$DATABASE_URL" up
   go run .
   ```

3. Check `GET http://localhost:8082/health`. Run [Identity](../identity/README.md) for admin requests.

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Catalog PostgreSQL connection string, required |
| `APP_PORT` | HTTP port, defaults to `8082` |
| `IDENTITY_BASE_URL` | Identity service URL, defaults to `http://localhost:8081` |
| `R2_ENDPOINT`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` | R2 bucket settings, required to upload and verify images |
| `R2_PUBLIC_BASE_URL` | Public HTTPS base URL for serving images. Image URLs are `null` when unset |

## API

Local base URL is `http://localhost:8082/api/v1/catalog`. Admin routes require an OWNER or ADMIN login token in the `Authorization: Bearer <token>` header.

| Method | Path | Access |
|---|---|---|
| GET | `/cars`, `/cars/{slug}`, `/cars/{slug}/related` | Public |
| GET | `/brands`, `/home/banners`, `/home/featured-cars` | Public |
| GET, POST | `/admin/brands` | OWNER/ADMIN |
| PATCH | `/admin/brands/{id}` | OWNER/ADMIN |
| GET, POST | `/admin/cars` | OWNER/ADMIN |
| GET, PATCH | `/admin/cars/{id}` | OWNER/ADMIN |
| GET | `/admin/banners` | OWNER/ADMIN |
| GET, PUT, DELETE | `/admin/banners/{position}` | OWNER/ADMIN |
| GET, PUT | `/admin/featured-cars` | OWNER/ADMIN |
| GET, POST | `/admin/cars/{id}/units` | OWNER/ADMIN |
| GET, PATCH | `/admin/units/{id}` | OWNER/ADMIN |
| POST | `/admin/images` | OWNER/ADMIN |

- Upload files through `/admin/images`, then submit the returned object keys with the Car, Brand, or Banner data. Creating a Car requires one cover and at least one gallery photo.
- Public lists and recommendations show active variants. An inactive variant linked to a Hero banner still has a public detail page with `status: inactive`.
- The Hero has three fixed slots. Save a slot with `car_id` and banner copy and image. Filled slots stay visible if the linked Car is inactive. Existing manual Brand, model, and slug inputs remain supported.

- Featured Cars accepts up to three Car IDs in display order. Inactive selections stay in the admin list and are skipped in the public list.

- Physical units have a unique case-insensitive internal code and default to inactive. Unit records appear only in admin APIs.

See [OpenAPI](../../packages/contracts/openapi/catalog.v1.yaml) for the contract and [DEVELOPMENT.md](DEVELOPMENT.md) for Bruno examples.

## Checks

```sh
go vet ./...
go test -race -count=1 ./...
go build ./...
```

Set `CATALOG_TEST_DATABASE_URL` to a dedicated PostgreSQL test database with permission to create schemas to enable integration tests. Identity and R2 are simulated in tests.
