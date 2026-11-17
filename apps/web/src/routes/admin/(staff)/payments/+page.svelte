<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { AdminPaymentPreview } from '#lib/types/admin-transactions.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { listAdminPaymentPreviews } from '#lib/utils/admin-transaction-preview.ts';
	import { paymentStatusStyles } from '#lib/utils/admin-transaction-status.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	function detailHref(id: string) {
		return resolve('/admin/(staff)/payments/[id]', { id }) + (page.shallow?.url ?? page.url).search;
	}

	const config: DataTableConfig<AdminPaymentPreview> = {
		queryKey: 'admin-payments-preview',
		loadPage: listAdminPaymentPreviews,
		label: 'Payments',
		rowKey: (row) => row.id,
		columns: [
			{ label: 'Payment', cell: reference },
			{ label: 'Booking', cell: booking },
			{ label: 'Bank', field: 'bank' },
			{ label: 'Amount', cell: amount },
			{ label: 'Payment status', cell: status },
			{ label: 'Reconciliation', cell: reconciliation },
			{ label: 'Updated at', cell: updated },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{
				name: 'search',
				label: 'Search',
				type: 'search',
				placeholder: 'Search IDs or provider reference'
			},
			{
				name: 'status',
				label: 'Payment status',
				type: 'select',
				placeholder: 'Select status',
				options: Object.entries(paymentStatusStyles).map(([value, style]) => ({
					value,
					label: style.label
				}))
			},
			{
				name: 'reconciliation',
				label: 'Reconciliation',
				type: 'select',
				placeholder: 'Select reconciliation',
				options: [
					{ value: 'required', label: 'Review required' },
					{ value: 'clear', label: 'No review' }
				]
			}
		],
		pageSize: 20,
		emptyMessage: 'No payment attempts yet.',
		filteredEmptyMessage: 'No payment attempts match your filters.'
	};
</script>

<svelte:head><title>Payments — SROOM</title></svelte:head>

<header class="mb-7">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Payments</h1>
	<p class="mt-2 text-sm text-ink-muted">
		Preview data for inspecting Virtual Account attempts and reconciliation cases.
	</p>
</header>

<DataTableGenerator {config} />

{#snippet reference(row: AdminPaymentPreview)}
	<Link
		href={detailHref(row.id)}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
		>{row.id}</Link
	>
{/snippet}

{#snippet booking(row: AdminPaymentPreview)}
	<Link
		href={resolve('/admin/(staff)/bookings/[id]', { id: row.bookingId })}
		variant="plain"
		class="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
		>{row.bookingId}</Link
	>
{/snippet}

{#snippet amount(row: AdminPaymentPreview)}
	<span class="font-medium whitespace-nowrap tabular-nums">{formatIDR(row.total)}</span>
{/snippet}

{#snippet status(row: AdminPaymentPreview)}
	<TransactionStatus value={{ payment: row.status }} />
{/snippet}

{#snippet reconciliation(row: AdminPaymentPreview)}
	{#if row.reviewReason}
		<span
			class="inline-flex rounded-full bg-information/10 px-2.5 py-1 text-xs font-semibold whitespace-nowrap text-information"
			>Review required</span
		>
	{:else}
		<span class="text-ink-muted">—</span>
	{/if}
{/snippet}

{#snippet updated(row: AdminPaymentPreview)}
	{@const [date, time] = formatWitaDateTime(row.updatedAt).split(', ')}
	<time datetime={row.updatedAt} class="block whitespace-nowrap tabular-nums">
		{date}
		<span class="block text-xs/5 text-ink-muted">{time} WITA</span>
	</time>
{/snippet}

{#snippet actions(row: AdminPaymentPreview)}
	<Button
		href={detailHref(row.id)}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View payment ${row.id}`}>View</Button
	>
{/snippet}
