# SROOM web

Customer and staff web application for SROOM, a self-drive car rental service in Samarinda. The app covers public vehicle discovery, customer accounts, checkout and BNI Virtual Account payment, booking history and invoices, plus staff fleet and transaction management.

## Features

### Public

- Landing page with Catalog banners, featured cars, and brand logos.
- Vehicle catalog with search, filters, sorting, and pagination.
- Vehicle detail with gallery, specifications, daily rate, and booking entry point.
- Rental terms and public account pages.

### Customer

- Register, sign in, restore a session, change password, and sign out.
- Checkout with WITA pickup/return scheduling and backend-calculated quotes.
- Create a 15-minute booking hold after accepting rental terms.
- Pay with a Midtrans sandbox BNI Virtual Account and check payment status.
- View saved bookings, booking detail, payment state, and printable invoices.
- My Bookings filters and pagination are preserved in the URL.

### Staff

- Separate ADMIN and OWNER sign-in.
- Manage brands, cars, physical units, banners, and featured cars.
- Review customers, bookings, payments, and issued invoices.
- Read Overview and Reports with WITA date ranges.
- OWNER can manage ADMIN accounts; ADMIN cannot.

## Tech stack

- SvelteKit 3 with Svelte 5 runes and TypeScript
- Tailwind CSS v4 and project-owned custom components
- Vite 8 and Bun
- TanStack Query for server state
- Axios for Gateway requests
- Bits UI for accessible select and combobox primitives
- Zod for validation
- Lucide icons and locally bundled Fontsource fonts
- Vitest, ESLint, and Prettier

## Requirements

- Bun
- A running SROOM Gateway at `http://localhost:8080`
- Identity, Catalog, Booking, and Payment services for their respective features
- Redis, RabbitMQ, and Booking/Payment workers for the complete sandbox payment flow

## Run locally

Run commands from `apps/web`.

1. Create `.env` from `env.example`.

   ```sh
   cp env.example .env
   ```

2. Set the Gateway URL.

   ```dotenv
   PUBLIC_GATEWAY_BASE_URL=http://localhost:8080
   ```

3. Install dependencies and start the development server.

   ```sh
   bun install
   bun run dev
   ```

4. Open `http://localhost:5173`.

The Gateway must allow the web origin. For local development, use the same hostname for web and Gateway even though the ports differ. Configure the exact origin in the Gateway's `WEB_ORIGINS` setting.

## Configuration

| Variable                  | Purpose                                                                                                |
| ------------------------- | ------------------------------------------------------------------------------------------------------ |
| `PUBLIC_GATEWAY_BASE_URL` | Gateway base URL used by browser requests. Defaults to the current hostname on port `8080` when empty. |

Customer and staff requests use separate HttpOnly session cookies. The browser calls only the Gateway; the Gateway forwards authenticated requests to the owning services.

## Search indexing

This app is configured as a private deployment. `static/robots.txt` disallows all crawlers, the global HTML shell sets a noindex robots meta tag, and the server hook adds `X-Robots-Tag` to every response. Do not add a sitemap or public indexing metadata.

## Scripts

| Command             | Purpose                              |
| ------------------- | ------------------------------------ |
| `bun run dev`       | Start the Vite development server    |
| `bun run build`     | Build the production app             |
| `bun run preview`   | Preview the production build locally |
| `bun run check`     | Run Svelte and TypeScript checks     |
| `bun run test`      | Run the Vitest suite once            |
| `bun run test:unit` | Start Vitest in watch mode           |
| `bun run lint`      | Check formatting and lint rules      |
| `bun run format`    | Format the project with Prettier     |

## Rental and payment notes

- Pickup starts tomorrow in WITA. Pickup and return times use WITA handover slots.
- Pricing is inclusive per calendar day and is calculated by the backend.
- A booking is confirmed only after a verified provider payment and the required Booking transition. A paid Payment is not enough by itself.
- The local payment flow uses the Midtrans sandbox BNI Virtual Account simulator. Set `BOOKING_DEMO_PAYMENTS=false` and keep the Booking and Payment consumers running.
- SMTP, email delivery, and a public webhook tunnel are optional for local frontend testing.
- Invoice printing uses the browser's Print / Save as PDF flow.

## Project structure

```text
apps/web/
├── src/
│   ├── routes/
│   │   ├── +page.*                 # landing page and shared public layout
│   │   ├── (auth)/                 # /login and /register (group adds no URL segment)
│   │   ├── cars/                   # /cars and /cars/[slug] catalog pages
│   │   ├── rental-terms/           # public rental policy
│   │   ├── checkout/[slug]/        # schedule, quote, hold, and reserve
│   │   ├── payment/[bookingId]/    # BNI VA instructions and payment status
│   │   ├── bookings/               # success, expired, and review outcome pages
│   │   ├── account/                # profile, password, bookings, detail, invoice
│   │   └── admin/                  # staff login and (staff) management pages
│   ├── lib/
│   │   ├── api/                    # Gateway clients for customer/staff/catalog/identity
│   │   ├── queries/                # TanStack Query options and cache keys
│   │   ├── types/                  # API contracts and browser view models
│   │   ├── utils/                  # formatting, filters, rental/report state, helpers
│   │   ├── validation/             # form validation rules
│   │   ├── components/
│   │   │   ├── layout/             # public header, footer, account navigation
│   │   │   ├── home/               # landing sections
│   │   │   ├── cars/               # catalog cards, gallery, rental summary
│   │   │   ├── auth/               # sign-in and registration forms
│   │   │   ├── booking/            # checkout, payment, summaries, invoice document
│   │   │   ├── admin/              # staff shell, tables, fleet and report components
│   │   │   └── ui/                 # shared buttons, fields, dialogs, data table
│   │   ├── ui/                     # toast integration
│   │   └── assets/                 # local visual assets such as the favicon
│   ├── app.html
│   ├── app.d.ts
│   └── routes/layout.css           # Tailwind theme and global styles
├── static/                         # public images, fonts, licenses, and metadata
├── env.example
├── package.json
├── vite.config.ts
├── tsconfig.json
├── eslint.config.js
└── prettier.config.js
```

Route groups such as `(auth)` and `(staff)` organize layouts without changing the public URL. Dynamic segments such as `[slug]`, `[bookingId]`, and `[id]` are loaded from the Gateway.

## Checks

```sh
bun run check
bun run test
bun run lint
bun run build
```

The hosting adapter is not finalized yet; the project currently uses SvelteKit's `adapter-auto`.
