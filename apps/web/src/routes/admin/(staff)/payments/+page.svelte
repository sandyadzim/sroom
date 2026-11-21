<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import { fetchStaffPaymentPage } from '#lib/api/staff-transactions.ts';
	import type { StaffPaymentRow } from '#lib/types/admin-transactions.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { paymentStatusStyles } from '#lib/utils/admin-transaction-status.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	function detailHref(id: string) {
		return resolve('/admin/(staff)/payments/[id]', { id }) + (page.shallow?.url ?? page.url).search;
	}

	const config: DataTableConfig<StaffPaymentRow> = {
		queryKey: 'staff-payments',
		loadPage: (params, signal) => fetchStaffPaymentPage(params, signal),
		label: 'Payments',
		rowKey: (row) => row.payment_id,
		columns: [
			{ label: 'Payment', cell: reference },
			{ label: 'Booking', cell: booking },
			{ label: 'Bank', cell: bank },
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
				placeholder: 'Search payment, booking or provider order ID'
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
		emptyMessage: 'No payments yet.',
		filteredEmptyMessage: 'No payments match your filters.'
	};
</script>

<svelte:head><title>Payments — SROOM</title></svelte:head>

<header class="mb-7">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Payments</h1>
</header>

<DataTableGenerator {config} />

{#snippet reference(row: StaffPaymentRow)}
	<Link
		href={detailHref(row.payment_id)}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
		>{row.payment_id}</Link
	>
{/snippet}

{#snippet booking(row: StaffPaymentRow)}
	<Link
		href={resolve('/admin/(staff)/bookings/[id]', { id: row.booking_id })}
		variant="plain"
		class="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
		>{row.booking_id}</Link
	>
{/snippet}

{#snippet bank(row: StaffPaymentRow)}
	{row.bank.toUpperCase()}
{/snippet}

{#snippet amount(row: StaffPaymentRow)}
	<span class="font-medium whitespace-nowrap tabular-nums">{formatIDR(row.amount_idr)}</span>
{/snippet}

{#snippet status(row: StaffPaymentRow)}
	<TransactionStatus value={{ payment: row.status }} />
{/snippet}

{#snippet reconciliation(row: StaffPaymentRow)}
	{#if row.review_required}
		<span
			class="inline-flex rounded-full bg-information/10 px-2.5 py-1 text-xs font-semibold whitespace-nowrap text-information"
			>Review required</span
		>
	{:else}
		<span class="text-ink-muted">—</span>
	{/if}
{/snippet}

{#snippet updated(row: StaffPaymentRow)}
	{@const [date, time] = formatWitaDateTime(row.updated_at).split(', ')}
	<time datetime={row.updated_at} class="block whitespace-nowrap tabular-nums">
		{date}
		<span class="block text-xs/5 text-ink-muted">{time} WITA</span>
	</time>
{/snippet}

{#snippet actions(row: StaffPaymentRow)}
	<Button
		href={detailHref(row.payment_id)}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View payment ${row.payment_id}`}>View</Button
	>
{/snippet}
