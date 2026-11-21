<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowRight } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import DailyReportChart from '#lib/components/admin/DailyReportChart.svelte';
	import ReportMetrics from '#lib/components/admin/ReportMetrics.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { getApiError } from '#lib/api/staff.ts';
	import { staffOverviewQuery } from '#lib/queries/staff-reports.ts';
	import { customerLookupQuery } from '#lib/queries/staff-transactions.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate, formatWitaDateTime } from '#lib/utils/date.ts';
	import { dailyChartData, fleetDetail } from '#lib/utils/staff-reports.ts';
	import { attachCustomerNames, uniqueCustomerIds } from '#lib/utils/staff-transactions.ts';

	const overview = createQuery(() => staffOverviewQuery());
	const customers = createQuery(() =>
		customerLookupQuery(overview.data ? uniqueCustomerIds(overview.data.recent_bookings) : [])
	);
	const recent = $derived(
		overview.data ? attachCustomerNames(overview.data.recent_bookings, customers.data ?? []) : []
	);
	const summary = $derived(overview.data?.summary);
	const metrics = $derived(
		overview.data && summary
			? [
					{ label: 'Bookings', value: summary.total_bookings },
					{
						label: 'Confirmed booking value',
						value: formatIDR(summary.confirmed_value_idr)
					},
					{
						label: 'Pending payment bookings',
						value: summary.pending_payment_bookings,
						detail: 'Bookings awaiting payment'
					},
					{
						label: 'Fleet cars',
						value: overview.data.fleet.cars.total,
						detail: fleetDetail(overview.data.fleet.cars)
					},
					{
						label: 'Physical units',
						value: overview.data.fleet.units.total,
						detail: fleetDetail(overview.data.fleet.units)
					}
				]
			: []
	);
	const reportsHref = $derived(
		overview.data
			? `${resolve('/admin/(staff)/reports')}?${new URLSearchParams({
					from: overview.data.period.from,
					to: overview.data.period.to
				})}`
			: resolve('/admin/(staff)/reports')
	);
</script>

<svelte:head><title>Overview — SROOM</title></svelte:head>

<header class="mb-6 flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Overview</h1>
		<p class="mt-2 text-sm leading-6 text-ink-muted">
			{#if overview.data}
				Simulated booking data · {formatWitaDate(overview.data.period.from)} – {formatWitaDate(
					overview.data.period.to
				)} WITA
			{:else if overview.isError}
				Report unavailable
			{:else}
				Loading current period…
			{/if}
		</p>
	</div>
	{#if overview.data}
		<Button href={reportsHref} variant="secondary">
			View Reports <ArrowRight size={16} aria-hidden="true" />
		</Button>
	{:else}
		<Button variant="secondary" disabled>
			View Reports <ArrowRight size={16} aria-hidden="true" />
		</Button>
	{/if}
</header>

{#if overview.isError}
	<section class="rounded-xl border border-line px-4 py-6 sm:px-5">
		<p role="alert" class="text-negative">
			{getApiError(overview.error)?.message ?? 'Could not load the overview.'}
		</p>
		<Button
			class="mt-4"
			variant="secondary"
			disabled={overview.isFetching}
			onclick={() => overview.refetch()}
		>
			{overview.isFetching ? 'Loading…' : 'Try again'}
		</Button>
	</section>
{:else if overview.isPending || !overview.data}
	<ReportMetrics items={[]} loading />
	<div class="mt-6 grid gap-4 xl:grid-cols-2" aria-hidden="true">
		{#each [0, 1] as index (index)}
			<div class="h-72 rounded-xl border border-line bg-panel p-4 sm:p-5">
				<div class="h-5 w-40 rounded bg-panel-muted motion-safe:animate-pulse"></div>
				<div class="mt-6 h-40 rounded bg-panel-muted motion-safe:animate-pulse"></div>
			</div>
		{/each}
	</div>
	<p role="status" class="sr-only">Loading overview…</p>
{:else}
	<ReportMetrics items={metrics} />

	<div class="mt-6 grid gap-4 xl:grid-cols-2">
		<DailyReportChart
			title="Daily bookings"
			data={dailyChartData(overview.data.daily, 'bookings')}
		/>
		<DailyReportChart
			title="Confirmed booking value"
			data={dailyChartData(overview.data.daily, 'confirmed_value_idr')}
			format={formatIDR}
			unit={250_000}
		/>
	</div>

	<section aria-labelledby="recent-title" class="mt-7">
		<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
			<h2 id="recent-title" class="text-lg font-semibold tracking-tight">Recent bookings</h2>
			<Link href={resolve('/admin/(staff)/bookings')} class="text-sm font-medium text-primary">
				View all bookings <ArrowRight size={16} aria-hidden="true" />
			</Link>
		</div>
		{#if recent.length}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex (the table scrolls horizontally) -->
			<div
				role="region"
				aria-label="Recent bookings"
				tabindex="0"
				class="overflow-x-auto rounded-xl border border-line focus-visible:-outline-offset-2"
			>
				<table class="w-full min-w-max border-collapse text-left text-sm">
					<thead class="border-b border-line bg-page">
						<tr>
							{#each ['Booking', 'Customer', 'Vehicle', 'Created at', 'Status', 'Total'] as label (label)}
								<th scope="col" class="px-4 py-3 font-semibold">{label}</th>
							{/each}
						</tr>
					</thead>
					<tbody class="divide-y divide-line">
						{#each recent as booking (booking.id)}
							<tr>
								<td class="px-4 py-3"
									><Link
										href={resolve('/admin/(staff)/bookings/[id]', { id: booking.id })}
										variant="plain"
										class="font-medium text-primary underline decoration-primary/30"
										>{booking.id}</Link
									></td
								>
								<td class="px-4 py-3 wrap-anywhere">
									{booking.customer_name ?? booking.customer_id}
								</td>
								<td class="px-4 py-3"
									><Link
										href={resolve('/admin/(staff)/cars/[id]', { id: booking.variant_id })}
										variant="plain"
										class="text-primary underline decoration-primary/30"
										>{booking.vehicle_label}</Link
									></td
								>
								<td class="px-4 py-3 whitespace-nowrap tabular-nums"
									><time datetime={booking.created_at}
										>{formatWitaDateTime(booking.created_at)} WITA</time
									></td
								>
								<td class="px-4 py-3"><TransactionStatus value={{ booking: booking.status }} /></td>
								<td class="px-4 py-3 whitespace-nowrap tabular-nums"
									>{formatIDR(booking.total_idr)}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<p class="rounded-lg border border-line px-4 py-5 text-sm text-ink-muted">
				No bookings for this period.
			</p>
		{/if}
	</section>
{/if}
