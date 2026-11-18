<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowRight } from '@lucide/svelte';
	import DailyReportChart from '#lib/components/admin/DailyReportChart.svelte';
	import ReportMetrics from '#lib/components/admin/ReportMetrics.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { getAdminReportPreview, reportPreviewPeriod } from '#lib/utils/admin-report-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate, formatWitaDateTime } from '#lib/utils/date.ts';

	const report = getAdminReportPreview();
	const metrics = [
		{ label: 'Bookings', value: report.total },
		{ label: 'Confirmed booking value', value: formatIDR(report.confirmedValue) },
		{ label: 'Pending payments', value: report.pendingPayments },
		{ label: 'Fleet models', value: report.fleet.models },
		{ label: 'Physical units', value: report.fleet.units }
	];
</script>

<svelte:head><title>Overview — SROOM</title></svelte:head>

<header class="mb-6 flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Overview</h1>
		<p class="mt-2 text-sm leading-6 text-ink-muted">
			Simulated data · {formatWitaDate(reportPreviewPeriod.from)} – {formatWitaDate(
				reportPreviewPeriod.to
			)} WITA
		</p>
	</div>
	<Button href={resolve('/admin/(staff)/reports')} variant="secondary">
		View Reports <ArrowRight size={16} aria-hidden="true" />
	</Button>
</header>

<ReportMetrics items={metrics} />

<div class="mt-6 grid gap-4 xl:grid-cols-2">
	<DailyReportChart
		title="Daily bookings"
		data={report.days.map(({ day, bookings }) => ({ day, value: bookings }))}
	/>
	<DailyReportChart
		title="Confirmed booking value"
		data={report.days}
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
	{#if report.recent.length}
		<p id="recent-table-help" class="mb-2 text-xs text-ink-muted xl:sr-only">
			Scroll horizontally to see all columns.
		</p>
		<!-- svelte-ignore a11y_no_noninteractive_tabindex (the table scrolls horizontally) -->
		<div
			role="region"
			aria-label="Recent bookings"
			aria-describedby="recent-table-help"
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
					{#each report.recent as booking (booking.id)}
						<tr>
							<td class="px-4 py-3"
								><Link
									href={resolve('/admin/(staff)/bookings/[id]', { id: booking.id })}
									variant="plain"
									class="font-medium text-primary underline decoration-primary/30"
									>{booking.id}</Link
								></td
							>
							<td class="px-4 py-3">{booking.customer}</td>
							<td class="px-4 py-3">{booking.model}</td>
							<td class="px-4 py-3 whitespace-nowrap tabular-nums"
								><time datetime={booking.createdAt}
									>{formatWitaDateTime(booking.createdAt)} WITA</time
								></td
							>
							<td class="px-4 py-3"><TransactionStatus value={{ booking: booking.status }} /></td>
							<td class="px-4 py-3 whitespace-nowrap tabular-nums">{formatIDR(booking.total)}</td>
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
