<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft, Check, Printer } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { bookingHistorySearch, readBookingFilters } from '#lib/utils/booking-history.ts';
	import {
		bookingPreviewPaymentReference,
		checkoutPreviewCustomer,
		readBookingPreview
	} from '#lib/utils/booking-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate, formatWitaDateTime } from '#lib/utils/date.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let ready = $state(false);
	const booking = $derived(ready ? readBookingPreview(data.bookingId) : undefined);
	const selection = $derived(readBookingFilters(page.url.searchParams));
	const detailHref = $derived(
		resolve(
			`/account/bookings/[bookingId]${bookingHistorySearch(selection.status, selection.page)}`,
			{
				bookingId: data.bookingId
			}
		)
	);
	const invoice = $derived(booking?.outcome === 'confirmed' ? booking : null);
	const invoiceNumber = $derived(`INV-${data.bookingId.slice('preview-'.length).toUpperCase()}`);
	const issueDate = $derived(invoice?.outcomeAt ? new Date(invoice.outcomeAt).toISOString() : null);

	onMount(() => {
		ready = true;
	});
</script>

<svelte:head>
	<title>{invoice ? invoiceNumber : 'Invoice'} — SROOM</title>
</svelte:head>

<div class="mb-5 flex flex-wrap items-center justify-between gap-3 print:hidden">
	<Link
		href={booking ? detailHref : resolve('/account/bookings')}
		variant="plain"
		class="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-ink-muted hover:text-ink"
	>
		<ArrowLeft size={16} aria-hidden="true" />
		{booking ? 'Back to booking' : 'My Bookings'}
	</Link>
	{#if invoice}
		<Button variant="secondary" onclick={() => window.print()}>
			<Printer size={17} aria-hidden="true" />
			Print / Save as PDF
		</Button>
	{/if}
</div>

{#if booking === undefined}
	<p role="status" class="text-sm text-ink-muted">Loading invoice…</p>
{:else if !invoice}
	<section class="rounded-card border border-line bg-panel p-5 sm:p-8">
		<h1 class="text-2xl font-bold tracking-tight">Invoice unavailable</h1>
		<p class="mt-2 text-sm leading-6 text-ink-muted">
			{booking
				? 'Invoices are available once a booking is confirmed.'
				: 'This invoice couldn’t be found. Return to My Bookings to view your reservations.'}
		</p>
	</section>
{:else}
	<article
		data-invoice-sheet
		aria-labelledby="invoice-heading"
		class="rounded-card border border-line bg-panel p-5 sm:p-8"
	>
		<header class="flex flex-wrap items-start justify-between gap-4 border-b border-line pb-5">
			<div>
				<p class="font-display text-xl font-extrabold tracking-tight text-primary">SROOM.</p>
				<h1 id="invoice-heading" class="mt-3 text-4xl font-bold tracking-tight">Invoice</h1>
			</div>
			<span
				class="inline-flex items-center gap-1.5 rounded-full bg-positive/10 px-3 py-1.5 text-sm font-semibold text-positive"
			>
				<Check size={16} aria-hidden="true" />
				Paid
			</span>
		</header>

		<dl class="grid grid-cols-1 gap-x-8 gap-y-4 border-b border-line py-5 sm:grid-cols-2">
			<div class="min-w-0">
				<dt class="text-sm text-ink-muted">Invoice number</dt>
				<dd class="mt-1 text-sm font-semibold wrap-anywhere">{invoiceNumber}</dd>
			</div>
			<div>
				<dt class="text-sm text-ink-muted">Issued on</dt>
				<dd class="mt-1 text-sm font-semibold">
					{#if issueDate}<time datetime={issueDate}>{formatWitaDate(issueDate)}</time>{:else}—{/if}
				</dd>
			</div>
			<div class="min-w-0">
				<dt class="text-sm text-ink-muted">Booking reference</dt>
				<dd class="mt-1 text-sm font-semibold wrap-anywhere">{data.bookingId.toUpperCase()}</dd>
			</div>
			<div>
				<dt class="text-sm text-ink-muted">Billed to</dt>
				<dd class="mt-1 text-sm font-semibold">{checkoutPreviewCustomer.name}</dd>
			</div>
		</dl>

		<section aria-labelledby="rental-heading" class="mt-5">
			<h2 id="rental-heading" class="text-lg font-bold">{invoice.carName} · {invoice.carTrim}</h2>
			<p class="mt-1 text-sm text-ink-muted">Samarinda, East Kalimantan</p>
			<dl class="mt-3 grid gap-3 text-sm sm:grid-cols-2">
				<div>
					<dt class="text-ink-muted">Pickup</dt>
					<dd class="mt-0.5 font-medium">
						<time datetime={invoice.pickupAt}>{formatWitaDateTime(invoice.pickupAt)} WITA</time>
					</dd>
				</div>
				<div>
					<dt class="text-ink-muted">Return</dt>
					<dd class="mt-0.5 font-medium">
						<time datetime={invoice.returnAt}>{formatWitaDateTime(invoice.returnAt)} WITA</time>
					</dd>
				</div>
			</dl>
		</section>

		<table class="mt-5 w-full border-collapse text-sm tabular-nums" aria-label="Rental charges">
			<thead class="border-y border-line bg-panel-muted">
				<tr>
					<th scope="col" class="px-3 py-2 text-left font-semibold">Description</th>
					<th scope="col" class="px-3 py-2 text-right font-semibold">Amount</th>
				</tr>
			</thead>
			<tbody>
				<tr class="border-b border-line align-top">
					<td class="px-3 py-3">
						<p class="font-medium">Daily rental</p>
						<p class="mt-0.5 text-ink-muted">
							{invoice.days}
							{invoice.days === 1 ? 'day' : 'days'} × {formatIDR(invoice.dailyRate)}
						</p>
					</td>
					<td class="px-3 py-3 text-right font-medium whitespace-nowrap"
						>{formatIDR(invoice.total)}</td
					>
				</tr>
			</tbody>
			<tfoot>
				<tr>
					<th scope="row" class="px-3 pt-3 text-left font-bold">Total paid</th>
					<td class="px-3 pt-3 text-right text-lg font-bold whitespace-nowrap"
						>{formatIDR(invoice.total)}</td
					>
				</tr>
			</tfoot>
		</table>

		<dl class="mt-5 grid gap-3 border-t border-line pt-4 text-sm sm:grid-cols-2">
			<div>
				<dt class="text-ink-muted">Payment method</dt>
				<dd class="mt-0.5 font-medium">Virtual Account</dd>
			</div>
			<div>
				<dt class="text-ink-muted">Payment reference</dt>
				<dd class="mt-0.5 font-medium wrap-anywhere">
					{bookingPreviewPaymentReference(data.bookingId)}
				</dd>
			</div>
		</dl>
		<p class="mt-4 text-xs text-ink-muted">Non-tax invoice</p>
	</article>
{/if}

<style>
	@media print {
		@page sroom-invoice {
			size: A4;
			margin: 14mm;
		}

		:global(body:has([data-invoice-sheet])) {
			page: sroom-invoice;
		}

		:global(body:has([data-invoice-sheet]) [data-header]),
		:global(body:has([data-invoice-sheet]) #footer),
		:global(body:has([data-invoice-sheet]) [data-account-shell] nav) {
			display: none;
		}

		:global(body:has([data-invoice-sheet]) .min-h-dvh),
		:global(body:has([data-invoice-sheet]) [data-account-grid]) {
			display: block;
			min-height: 0;
		}

		:global(body:has([data-invoice-sheet]) [data-account-shell]) {
			max-width: none;
			padding: 0;
		}

		[data-invoice-sheet] {
			border: 0;
			border-radius: 0;
			padding: 0;
		}

		[data-invoice-sheet] header,
		[data-invoice-sheet] dl,
		[data-invoice-sheet] table {
			break-inside: avoid;
		}
	}
</style>
