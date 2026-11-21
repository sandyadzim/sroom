# SROOM Gateway

Local Go + Gin gateway for Identity, Catalog, Booking, and Payment. Use `http://localhost:8080` as the base URL.

## Run locally

1. Start [Identity](../identity/README.md) on `8081` and [Catalog](../catalog/README.md) on `8082`.
2. Run `go run .` from `services/gateway`.
3. Check `GET http://localhost:8080/health` for `{"status":"ok"}`. This checks the gateway process.

| Environment variable | Default |
|---|---|
| `APP_ENV` | `development` (any other value sets `Secure` on the staff cookie) |
| `APP_PORT` | `8080` |
| `IDENTITY_URL` | `http://localhost:8081` |
| `CATALOG_URL` | `http://localhost:8082` |
| `BOOKING_URL` | `http://localhost:8083` |
| `PAYMENT_URL` | `http://localhost:8084` |
| `WEB_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` |

Export variables before starting the gateway to override the defaults. It listens on IPv4 loopback. Identity still applies its login and registration rate limits, and requests through the gateway share its IP quota.

The staff browser calls the gateway directly. Set `WEB_ORIGINS` to a comma-separated list of exact frontend origins if you use a different host or port. The gateway answers browser preflights and allows the Authorization and Content-Type headers for those origins. Identity and Catalog still enforce their own authorization.

## Staff browser session

`POST /api/v1/staff/login` uses the existing Identity login and profile endpoints. Gateway accepts OWNER or ADMIN, returns the profile, and sets the `sroom_staff_session` HttpOnly cookie. The frontend sends it with `withCredentials: true`. Gateway converts it to a Bearer header for `/staff/me`, `/staff/logout`, `/staff/change-password`, `/staff/admins`, `/staff/customers`, and `/staff/catalog/admin/*` under `/api/v1`. Identity and Catalog still enforce role permissions. Successful logout revokes the session and expires the cookie.

Staff mutations require an exact origin in `WEB_ORIGINS`. For local development, use the same hostname for frontend and API, for example `localhost:5173` and `localhost:8080`. On HTTPS, set `APP_ENV=production` and serve both on the same site so the `SameSite=Lax` cookie is sent.

## Try in Bruno

Set `base_url` to `http://localhost:8080` in your existing requests.

1. `POST {{base_url}}/api/v1/identity/login` with your local account credentials.
2. `GET {{base_url}}/api/v1/identity/me` with the login token under **Auth → Bearer Token**.
3. `GET {{base_url}}/api/v1/catalog/cars?page=1&limit=10` to browse without a token.
4. `POST {{base_url}}/api/v1/catalog/admin/images` with an OWNER or ADMIN token. In **Body → Multipart Form**, add a `file` field and select a JPEG, PNG, or WebP. Catalog needs R2 configuration for this request. A CUSTOMER token returns `403`.

Set `X-Request-ID` to `bruno-check-1` to match the response with a service error log. If an upstream is stopped, gateway requests to it return `503`. Requests that exceed the upstream deadline return `504`. The other service and gateway health remain available.

Run `go test -race ./...` from `services/gateway`. See the [OpenAPI contract](../../packages/contracts/openapi/gateway.v1.yaml).

`GET /api/v1/staff/overview` combines Booking statistics and current Catalog fleet counts. `GET /api/v1/staff/reports` forwards optional `from`/`to` WITA creation dates to Booking. Both use the staff cookie; owning services verify ADMIN/OWNER. No new configuration is needed.

## Customer browser authentication

The public `/api/v1/identity/register` route creates a customer account without a session. The customer frontend uses `POST /api/v1/customer/login`, `GET /api/v1/customer/me`, and `POST /api/v1/customer/logout`. These session routes use a dedicated HttpOnly `sroom_customer_session` cookie scoped to `/api/v1/customer` and check the CUSTOMER role through Identity. The cookie expires with the Identity session. Use the same site for web and Gateway and configure the exact web origin in `WEB_ORIGINS`. `APP_ENV=production` requires HTTPS for cookies.

Customer Profile reads `GET /api/v1/customer/me`. The customer changes their own password with `POST /api/v1/customer/change-password` using their customer cookie and an allowed origin. The Gateway forwards the request to Identity's CUSTOMER-only password route. A successful update returns 204 and preserves existing sessions.

Checkout sends `POST /api/v1/customer/checkout/quotes` with the same customer cookie and an allowed origin. Gateway forwards the session as a Bearer token to [Booking](../booking/README.md).

`POST /api/v1/customer/bookings` uses the same cookie and origin. Gateway also forwards the required `Idempotency-Key` header.

`POST /api/v1/customer/bookings/{id}/payments/va` forwards the same customer session, Origin checks, and idempotency key to [Payment](../payment/README.md), with a 15-second upstream deadline. Send JSON `{}`; Payment determines BNI, amount, and deadline server-side. Direct Payment and private service routes are not exposed.

`GET /api/v1/customer/bookings/{id}/payment` uses the customer cookie. Staff use `GET /api/v1/staff/payments` and `GET /api/v1/staff/payments/{id}`. Gateway forwards these to Payment; Payment verifies roles and ownership. GET requires no Origin header, forwards list filters, and reads stored state without a Midtrans call.

`POST /api/v1/customer/bookings/{id}/payment/refresh` requires the customer cookie and an allowed Origin. Send `{}` without an idempotency key. Gateway forwards it to Payment with the existing 15-second deadline; Payment verifies ownership and looks up the saved provider order.

`POST /webhooks/midtrans` forwards provider notifications unchanged to Payment, with a 16 KiB body limit and a 10-second upstream deadline. It requires no browser cookie or Origin. Payment verifies the signature and retrieves the provider status before recording any outcome.

For booking history, customers use `GET /api/v1/customer/bookings` and `GET /api/v1/customer/bookings/{id}`. Staff use the same paths under `/api/v1/staff` with the staff session cookie. Booking checks the session role and customer ownership.

Invoice reads use `GET /api/v1/customer/bookings/{id}/invoice` and `GET /api/v1/staff/bookings/{id}/invoice` with their respective cookies. Gateway forwards to Payment, which enforces role/ownership. Invoice JSON is immutable after issuance; GET does not create it. Booking invoice-context stays private.
