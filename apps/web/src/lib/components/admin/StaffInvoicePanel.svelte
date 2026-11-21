<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { Printer } from '@lucide/svelte';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import InvoiceDocument from '#lib/components/booking/InvoiceDocument.svelte';
	import { staffInvoiceQuery } from '#lib/queries/staff-transactions.ts';
	import { invoiceFailure } from '#lib/utils/staff-transactions.ts';

	let { bookingId }: { bookingId: string } = $props();
	let open = $state(false);
	const invoice = createQuery(() => ({
		...staffInvoiceQuery(bookingId),
		enabled: open
	}));
	const failure = $derived(invoice.error ? invoiceFailure(invoice.error) : undefined);
</script>

<section
	aria-labelledby="invoice-panel-title"
	class="mt-6 border-t border-line pt-6 print:mt-0 print:border-0 print:pt-0"
>
	<div class="flex flex-wrap items-center justify-between gap-3 print:hidden">
		<div>
			<h2 id="invoice-panel-title" class="text-lg/7 font-semibold">Invoice</h2>
			<p class="mt-1 text-sm text-ink-muted">
				The issued invoice becomes available after a verified payment confirms the booking.
			</p>
		</div>
		<Button variant="secondary" onclick={() => (open = !open)}>
			{open ? 'Hide invoice' : 'View Invoice'}
		</Button>
	</div>

	{#if open}
		<div class="mt-4">
			{#if invoice.isPending}
				<p role="status" class="rounded-xl border border-line px-4 py-6 text-sm text-ink-muted">
					Loading invoice…
				</p>
			{:else if failure === 'not_available'}
				<div class="rounded-xl border border-line px-4 py-6 sm:px-5">
					<h3 class="text-base font-semibold">Invoice is being prepared</h3>
					<p role="status" class="mt-2 text-sm text-ink-muted">
						The booking is confirmed. Try again shortly to retrieve the invoice.
					</p>
					<Button
						class="mt-4"
						variant="secondary"
						disabled={invoice.isFetching}
						onclick={() => void invoice.refetch()}
					>
						{invoice.isFetching ? 'Loading…' : 'Try again'}
					</Button>
				</div>
			{:else if failure === 'not_found'}
				<p role="alert" class="rounded-xl border border-line px-4 py-6 text-sm text-negative">
					Invoice not found for this booking.
				</p>
			{:else if failure === 'error' || !invoice.data}
				<div class="rounded-xl border border-line px-4 py-6 sm:px-5">
					<p role="alert" class="text-sm text-negative">
						{getApiError(invoice.error)?.message ?? 'Could not load this invoice.'}
					</p>
					<Button
						class="mt-4"
						variant="secondary"
						disabled={invoice.isFetching}
						onclick={() => void invoice.refetch()}
					>
						{invoice.isFetching ? 'Loading…' : 'Try again'}
					</Button>
				</div>
			{:else}
				<div class="mb-3 flex justify-end print:hidden">
					<Button variant="secondary" onclick={() => window.print()}>
						<Printer size={17} aria-hidden="true" />Print / Save as PDF
					</Button>
				</div>
				<InvoiceDocument invoice={invoice.data} />
			{/if}
		</div>
	{/if}
</section>
