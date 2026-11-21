<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import { fetchStaffBookingPage } from '#lib/api/staff-transactions.ts';
	import type { StaffBookingRow } from '#lib/types/admin-transactions.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { bookingStatusStyles } from '#lib/utils/admin-transaction-status.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	function detailHref(id: string) {
		return resolve('/admin/(staff)/bookings/[id]', { id }) + (page.shallow?.url ?? page.url).search;
	}

	const config: DataTableConfig<StaffBookingRow> = {
		queryKey: 'staff-bookings',
		loadPage: (params, signal) => fetchStaffBookingPage(params, signal),
		label: 'Bookings',
		rowKey: (row) => row.id,
		columns: [
			{ label: 'Booking', cell: reference },
			{ label: 'Customer', cell: customer },
			{ label: 'Vehicle', cell: vehicle },
			{ label: 'Pickup', cell: pickup },
			{ label: 'Return', cell: returnAt },
			{ label: 'Total', cell: total },
			{ label: 'Booking status', cell: status },
			{ label: 'Created at', cell: created },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{
				name: 'search',
				label: 'Search',
				type: 'search',
				placeholder: 'Search booking ID, customer ID or vehicle'
			},
			{
				name: 'status',
				label: 'Booking status',
				type: 'select',
				placeholder: 'Select status',
				options: Object.entries(bookingStatusStyles).map(([value, style]) => ({
					value,
					label: style.label
				}))
			},
			{
				name: 'variant_id',
				label: 'Vehicle',
				type: 'car',
				placeholder: 'Search by make, model, or trim'
			},
			{ name: 'pickup_date', label: 'Pickup date', type: 'date' }
		],
		pageSize: 20,
		emptyMessage: 'No bookings yet.',
		filteredEmptyMessage: 'No bookings match your filters.'
	};
</script>

<svelte:head><title>Bookings — SROOM</title></svelte:head>

<header class="mb-7">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Bookings</h1>
</header>

<DataTableGenerator {config} />

{#snippet reference(row: StaffBookingRow)}
	<Link
		href={detailHref(row.id)}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
	>
		{row.id}
	</Link>
{/snippet}

{#snippet customer(row: StaffBookingRow)}
	<span class="font-medium">{row.customer_name ?? row.customer_id}</span>
{/snippet}

{#snippet vehicle(row: StaffBookingRow)}
	<Link
		href={resolve('/admin/(staff)/cars/[id]', { id: row.variant_id })}
		variant="plain"
		class="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
	>
		{row.vehicle_label}
	</Link>
{/snippet}

{#snippet pickup(row: StaffBookingRow)}
	{@render dateTime(row.pickup_at)}
{/snippet}

{#snippet returnAt(row: StaffBookingRow)}
	{@render dateTime(row.return_at)}
{/snippet}

{#snippet total(row: StaffBookingRow)}
	<span class="font-medium whitespace-nowrap tabular-nums">{formatIDR(row.total_idr)}</span>
{/snippet}

{#snippet status(row: StaffBookingRow)}
	<TransactionStatus value={{ booking: row.status }} />
{/snippet}

{#snippet created(row: StaffBookingRow)}
	{@render dateTime(row.created_at)}
{/snippet}

{#snippet dateTime(value: string)}
	{@const [date, time] = formatWitaDateTime(value).split(', ')}
	<time datetime={value} class="block whitespace-nowrap tabular-nums">
		{date}
		<span class="block text-xs/5 text-ink-muted">{time} WITA</span>
	</time>
{/snippet}

{#snippet actions(row: StaffBookingRow)}
	<Button
		href={detailHref(row.id)}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View booking ${row.id}`}>View</Button
	>
{/snippet}
