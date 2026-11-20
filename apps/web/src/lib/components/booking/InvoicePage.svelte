<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import { ArrowLeft, Printer } from '@lucide/svelte';
	import { invoiceQuery, bookingQuery } from '#lib/queries/rental.ts';
	import { getApiError } from '#lib/api/error.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import CustomerRequestState from './CustomerRequestState.svelte';
	import InvoiceDocument from './InvoiceDocument.svelte';
	let { customerId, bookingId }: { customerId: string; bookingId: string } = $props();
	const invoice = createQuery(() => invoiceQuery(customerId, bookingId));
	const notAvailable = $derived(getApiError(invoice.error)?.code === 'INVOICE_NOT_AVAILABLE');
	const booking = createQuery(() => ({
		...bookingQuery(customerId, bookingId),
		enabled: notAvailable
	}));
</script>

<svelte:head><title>{invoice.data?.invoice_number ?? 'Invoice'} — SROOM</title></svelte:head>
<div class="mb-5 flex flex-wrap items-center justify-between gap-3 print:hidden">
	<Link
		href={resolve('/account/bookings/[bookingId]', { bookingId })}
		variant="plain"
		class="inline-flex min-h-11 items-center gap-2 text-sm text-ink-muted"
		><ArrowLeft size={16} aria-hidden="true" />Back to booking</Link
	>
	{#if invoice.data && !invoice.isError}<Button variant="secondary" onclick={() => window.print()}
			><Printer size={17} aria-hidden="true" />Print / Save as PDF</Button
		>{/if}
</div>
{#if notAvailable}
	{#if booking.isError}<CustomerRequestState
			error={booking.error}
			label="booking"
			retry={() => void booking.refetch()}
		/>
	{:else if !booking.data}<CustomerRequestState loading label="booking" />
	{:else}<section class="space-y-4 py-6">
			<h1 class="text-2xl font-bold">
				{booking.data.status === 'confirmed' ? 'Invoice is being prepared' : 'Invoice unavailable'}
			</h1>
			<p role="status" class="text-sm text-ink-muted">
				{booking.data.status === 'confirmed'
					? 'Your booking is confirmed. Try again shortly to retrieve the invoice.'
					: 'An invoice becomes available after your booking is confirmed.'}
			</p>
			<Button
				variant="secondary"
				disabled={invoice.isFetching}
				onclick={() => void invoice.refetch()}>Try again</Button
			>
		</section>{/if}
{:else if invoice.isError || !invoice.data}<CustomerRequestState
		error={invoice.error}
		loading={invoice.isPending}
		label="invoice"
		retry={() => void invoice.refetch()}
	/>
{:else}<InvoiceDocument invoice={invoice.data} />{/if}
