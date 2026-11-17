<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { AdminBookingPreview } from '#lib/types/admin-transactions.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { bookingModels, listAdminBookingPreviews } from '#lib/utils/admin-transaction-preview.ts';
	import { bookingStatusStyles } from '#lib/utils/admin-transaction-status.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	function detailHref(id: string) {
		return resolve('/admin/(staff)/bookings/[id]', { id }) + (page.shallow?.url ?? page.url).search;
	}

	const config: DataTableConfig<AdminBookingPreview> = {
		queryKey: 'admin-bookings-preview',
		loadPage: listAdminBookingPreviews,
		label: 'Bookings',
		rowKey: (row) => row.id,
		columns: [
			{ label: 'Booking', cell: reference },
			{ label: 'Customer', field: 'customer' },
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
				placeholder: 'Search booking, customer or vehicle'
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
				name: 'model',
				label: 'Vehicle',
				type: 'select',
				placeholder: 'Select vehicle',
				options: bookingModels
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
	<p class="mt-2 text-sm text-ink-muted">
		Preview data for inspecting reservations, schedules, and assigned units.
	</p>
</header>

<DataTableGenerator {config} />

{#snippet reference(row: AdminBookingPreview)}
	<Link
		href={detailHref(row.id)}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
	>
		{row.id}
	</Link>
{/snippet}

{#snippet vehicle(row: AdminBookingPreview)}
	<p>{row.model}</p>
	<p class="text-xs/5 text-ink-muted">{row.unit ?? 'Unassigned'}</p>
{/snippet}

{#snippet pickup(row: AdminBookingPreview)}
	{@render dateTime(row.pickupAt)}
{/snippet}

{#snippet returnAt(row: AdminBookingPreview)}
	{@render dateTime(row.returnAt)}
{/snippet}

{#snippet total(row: AdminBookingPreview)}
	<span class="font-medium whitespace-nowrap tabular-nums">{formatIDR(row.total)}</span>
{/snippet}

{#snippet status(row: AdminBookingPreview)}
	<TransactionStatus value={{ booking: row.status }} />
{/snippet}

{#snippet created(row: AdminBookingPreview)}
	{@render dateTime(row.createdAt)}
{/snippet}

{#snippet dateTime(value: string)}
	{@const [date, time] = formatWitaDateTime(value).split(', ')}
	<time datetime={value} class="block whitespace-nowrap tabular-nums">
		{date}
		<span class="block text-xs/5 text-ink-muted">{time} WITA</span>
	</time>
{/snippet}

{#snippet actions(row: AdminBookingPreview)}
	<Button
		href={detailHref(row.id)}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View booking ${row.id}`}>View</Button
	>
{/snippet}
