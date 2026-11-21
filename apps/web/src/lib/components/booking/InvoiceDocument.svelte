<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Invoice } from '#lib/types/payment.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate, formatWitaDateTime } from '#lib/utils/date.ts';
	let { invoice }: { invoice: Invoice } = $props();
	const issueDate = $derived(invoice.issued_at);
</script>

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
			<dd class="mt-1 text-sm font-semibold wrap-anywhere">{invoice.invoice_number}</dd>
		</div>
		<div>
			<dt class="text-sm text-ink-muted">Issued on</dt>
			<dd class="mt-1 text-sm font-semibold">
				{#if issueDate}<time datetime={issueDate}>{formatWitaDate(issueDate)}</time>{:else}—{/if}
			</dd>
		</div>
		<div class="min-w-0">
			<dt class="text-sm text-ink-muted">Booking reference</dt>
			<dd class="mt-1 text-sm font-semibold wrap-anywhere">{invoice.booking_id}</dd>
		</div>
		<div>
			<dt class="text-sm text-ink-muted">Billed to</dt>
			<dd class="mt-1 text-sm font-semibold wrap-anywhere">
				{invoice.billed_to.full_name ||
					invoice.billed_to.email}{#if invoice.billed_to.full_name}<span
						class="mt-1 block font-normal text-ink-muted">{invoice.billed_to.email}</span
					>{/if}
			</dd>
		</div>
		<div>
			<dt class="text-sm text-ink-muted">Confirmed on</dt>
			<dd class="mt-1 text-sm font-semibold">
				<time datetime={invoice.confirmed_at}>{formatWitaDateTime(invoice.confirmed_at)} WITA</time>
			</dd>
		</div>
	</dl>

	<section aria-labelledby="rental-heading" class="mt-5">
		<h2 id="rental-heading" class="text-lg font-bold">{invoice.vehicle_label}</h2>

		<dl class="mt-3 grid gap-3 text-sm sm:grid-cols-2">
			<div>
				<dt class="text-ink-muted">Pickup</dt>
				<dd class="mt-0.5 font-medium">
					<time datetime={invoice.pickup_at}>{formatWitaDateTime(invoice.pickup_at)} WITA</time>
				</dd>
			</div>
			<div>
				<dt class="text-ink-muted">Return</dt>
				<dd class="mt-0.5 font-medium">
					<time datetime={invoice.return_at}>{formatWitaDateTime(invoice.return_at)} WITA</time>
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
						{invoice.rental_days}
						{invoice.rental_days === 1 ? 'day' : 'days'} × {formatIDR(invoice.daily_rate_idr)}
					</p>
				</td>
				<td class="px-3 py-3 text-right font-medium whitespace-nowrap"
					>{formatIDR(invoice.total_idr)}</td
				>
			</tr>
		</tbody>
		<tfoot>
			<tr>
				<th scope="row" class="px-3 pt-3 text-left font-bold">Total paid</th>
				<td class="px-3 pt-3 text-right text-lg font-bold whitespace-nowrap"
					>{formatIDR(invoice.total_idr)}</td
				>
			</tr>
		</tfoot>
	</table>

	<dl class="mt-5 grid gap-3 border-t border-line pt-4 text-sm sm:grid-cols-2">
		<div>
			<dt class="text-ink-muted">Payment method</dt>
			<dd class="mt-0.5 font-medium">BNI Virtual Account</dd>
		</div>
		<div>
			<dt class="text-ink-muted">Payment reference</dt>
			<dd class="mt-0.5 font-medium wrap-anywhere">
				{invoice.provider_order_id}
			</dd>
		</div>
	</dl>
	<p class="mt-4 text-xs text-ink-muted">Non-tax invoice</p>
</article>

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

		:global(body:has([data-invoice-sheet]) [data-staff-sidebar]),
		:global(body:has([data-invoice-sheet]) [data-staff-header]) {
			display: none;
		}

		:global(body:has([data-invoice-sheet]) [data-staff-shell]) {
			display: block;
			min-height: 0;
		}

		:global(body:has([data-invoice-sheet]) [data-staff-shell] > div) {
			display: block;
			min-height: 0;
			padding: 0;
		}

		:global(body:has([data-invoice-sheet]) [data-staff-main]) {
			border: 0;
			border-radius: 0;
			background: transparent;
		}

		:global(body:has([data-invoice-sheet]) [data-staff-main] > div) {
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
