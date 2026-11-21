<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import DailyReportChart from '#lib/components/admin/DailyReportChart.svelte';
	import ReportMetrics from '#lib/components/admin/ReportMetrics.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { getApiError } from '#lib/api/staff.ts';
	import { staffReportsQuery } from '#lib/queries/staff-reports.ts';
	import type { BookingStatus } from '#lib/types/admin-transactions.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate } from '#lib/utils/date.ts';
	import {
		dailyChartData,
		formatConversion,
		parseReportRange,
		reportPeriodFor,
		validateReportRange
	} from '#lib/utils/staff-reports.ts';

	const rangeState = $derived(parseReportRange(page.url.searchParams));
	const period = $derived(reportPeriodFor(rangeState));
	const report = createQuery(() => ({
		...staffReportsQuery(period),
		enabled: rangeState.kind !== 'invalid'
	}));
	const summary = $derived(report.data?.summary);
	const metrics = $derived(
		summary
			? [
					{ label: 'Bookings', value: summary.total_bookings },
					{ label: 'Confirmed booking value', value: formatIDR(summary.confirmed_value_idr) },
					{ label: 'Payment conversion', value: formatConversion(summary.payment_conversion) },
					{
						label: 'Pending payment bookings',
						value: summary.pending_payment_bookings,
						detail: 'Bookings awaiting payment'
					},
					{ label: 'Expired bookings', value: summary.expired_bookings }
				]
			: []
	);
	const statuses: BookingStatus[] = ['confirmed', 'pending_payment', 'payment_review', 'expired'];
	const statusCounts = $derived(
		summary
			? {
					confirmed: summary.confirmed_bookings,
					pending_payment: summary.pending_payment_bookings,
					payment_review: summary.payment_review_bookings,
					expired: summary.expired_bookings
				}
			: undefined
	);

	let startDate = $state('');
	let endDate = $state('');
	let startError = $state<string>();
	let endError = $state<string>();
	let rangeError = $state<string>();

	$effect(() => {
		const state = rangeState;
		if (state.kind === 'invalid') {
			startDate = state.from;
			endDate = state.to;
			startError = state.errors.from;
			endError = state.errors.to;
			rangeError = state.errors.range ?? state.errors.query;
		} else if (state.kind === 'range') {
			startDate = state.period.from;
			endDate = state.period.to;
			startError = undefined;
			endError = undefined;
			rangeError = undefined;
		} else if (report.data) {
			startDate = report.data.period.from;
			endDate = report.data.period.to;
			startError = undefined;
			endError = undefined;
			rangeError = undefined;
		}
	});

	function applyDateRange(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		const errors = validateReportRange(startDate, endDate);
		startError = errors.from;
		endError = errors.to;
		rangeError = errors.range;
		if (errors.from || errors.to || errors.range) {
			event.currentTarget
				.querySelector<HTMLInputElement>(`[name=${errors.from ? 'from' : 'to'}]`)
				?.focus();
			return;
		}
		const query = new URLSearchParams({ from: startDate, to: endDate });
		void goto(resolve(`/admin/(staff)/reports?${query}`), { reset: false });
	}

	function resetDateRange() {
		startError = undefined;
		endError = undefined;
		rangeError = undefined;
		void goto(resolve('/admin/(staff)/reports'), { reset: false });
	}
</script>

<svelte:head><title>Reports — SROOM</title></svelte:head>

<header class="mb-6">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Reports</h1>
	<p class="mt-2 text-sm leading-6 text-ink-muted">
		Simulated booking figures by booking creation date in WITA.
	</p>
</header>

<form
	method="get"
	novalidate
	onsubmit={applyDateRange}
	class="mb-3 grid gap-4 rounded-xl border border-line bg-page p-4 sm:grid-cols-2 sm:p-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
>
	<TextField
		name="from"
		label="Start date"
		type="date"
		bind:value={startDate}
		error={startError}
		oninput={() => (startError = undefined)}
		required
	/>
	<TextField
		name="to"
		label="End date"
		type="date"
		bind:value={endDate}
		error={endError}
		oninput={() => (endError = undefined)}
		required
	/>
	<div class="flex flex-wrap items-center gap-2 sm:col-span-2 lg:col-span-1 lg:self-start lg:pt-7">
		<Button type="submit">Apply</Button>
		<Button variant="ghost" onclick={resetDateRange}>Reset</Button>
	</div>
</form>
{#if rangeError}
	<p
		role="alert"
		class="mb-5 rounded-lg border border-negative/20 bg-negative/5 px-4 py-3 text-sm text-negative"
	>
		{rangeError}
	</p>
{/if}

{#if rangeState.kind === 'invalid'}
	<section class="rounded-xl border border-line px-4 py-6 sm:px-5">
		<p class="text-sm text-ink-muted">
			The date range in the address bar is invalid. Fix the dates above or reset them to use the
			server default.
		</p>
		<Button class="mt-4" variant="secondary" onclick={resetDateRange}>Use default period</Button>
	</section>
{:else if report.isError}
	<section class="rounded-xl border border-line px-4 py-6 sm:px-5">
		<p role="alert" class="text-negative">
			{getApiError(report.error)?.message ?? 'Could not load the report.'}
		</p>
		<Button
			class="mt-4"
			variant="secondary"
			disabled={report.isFetching}
			onclick={() => report.refetch()}
		>
			{report.isFetching ? 'Loading…' : 'Try again'}
		</Button>
	</section>
{:else if report.isPending || !report.data}
	<ReportMetrics items={[]} loading />
	<div class="mt-6 grid items-start gap-4 xl:grid-cols-2" aria-hidden="true">
		{#each [0, 1] as index (index)}
			<div class="h-72 rounded-xl border border-line bg-panel p-4 sm:p-5">
				<div class="h-5 w-40 rounded bg-panel-muted motion-safe:animate-pulse"></div>
				<div class="mt-6 h-40 rounded bg-panel-muted motion-safe:animate-pulse"></div>
			</div>
		{/each}
	</div>
	<p role="status" class="sr-only">Loading report…</p>
{:else}
	<p role="status" class="mb-3 text-sm text-ink-muted">
		{formatWitaDate(report.data.period.from)} – {formatWitaDate(report.data.period.to)} · {summary?.total_bookings ??
			0}
		{summary?.total_bookings === 1 ? 'booking' : 'bookings'}
	</p>
	<ReportMetrics items={metrics} />

	<div class="mt-6 grid items-start gap-4 xl:grid-cols-2">
		<DailyReportChart
			title="Confirmed booking value by day"
			data={dailyChartData(report.data.daily, 'confirmed_value_idr')}
			format={formatIDR}
			unit={250_000}
		/>
		<section
			aria-labelledby="booking-status-title"
			class="rounded-xl border border-line p-4 sm:p-5"
		>
			<h2 id="booking-status-title" class="text-lg font-semibold tracking-tight">
				Bookings by status
			</h2>
			{#if summary && summary.total_bookings > 0 && statusCounts}
				<ul class="mt-4 space-y-3" role="list">
					{#each statuses as status (status)}
						<li>
							<div class="flex items-center justify-between gap-3">
								<TransactionStatus value={{ booking: status }} />
								<span class="text-sm font-medium tabular-nums">{statusCounts[status]}</span>
							</div>
							<div
								aria-hidden="true"
								class="mt-2 h-1.5 overflow-hidden rounded-full bg-panel-muted"
							>
								<div
									class="h-full rounded-full bg-primary"
									style:width={`${(100 * statusCounts[status]) / summary.total_bookings}%`}
								></div>
							</div>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="mt-5 rounded-lg bg-panel-muted px-4 py-5 text-sm text-ink-muted">
					No bookings for this period.
				</p>
			{/if}
		</section>
	</div>

	<section
		aria-labelledby="top-vehicles-title"
		class="mt-6 rounded-xl border border-line p-4 sm:p-5"
	>
		<h2 id="top-vehicles-title" class="text-lg font-semibold tracking-tight">
			Most-booked vehicles
		</h2>
		{#if report.data.top_variants.length}
			<table class="mt-3 w-full border-collapse text-left text-sm">
				<thead class="border-b border-line text-ink-muted">
					<tr
						><th scope="col" class="py-2 font-medium">Vehicle</th><th
							scope="col"
							class="py-2 text-right font-medium">Bookings</th
						></tr
					>
				</thead>
				<tbody class="divide-y divide-line">
					{#each report.data.top_variants as variant (variant.variant_id)}
						<tr>
							<th scope="row" class="py-2.5 font-medium">
								<a
									href={resolve('/admin/(staff)/cars/[id]', { id: variant.variant_id })}
									class="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
								>
									{variant.vehicle_label}
								</a>
							</th>
							<td class="py-2.5 text-right tabular-nums">{variant.bookings}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="mt-3 text-sm text-ink-muted">No vehicles to rank for this period.</p>
		{/if}
	</section>

	<details class="mt-5 rounded-lg border border-line px-4 py-2 text-sm">
		<summary class="cursor-pointer rounded-sm py-2 font-medium text-primary"
			>How metrics are counted</summary
		>
		<ul class="space-y-2 pb-3 text-ink-muted">
			<li>
				<strong class="font-medium text-ink">Confirmed booking value.</strong> Sum of booking snapshots
				for confirmed bookings with a verified Payment reference. This is booking value, not live cash
				revenue.
			</li>
			<li>
				<strong class="font-medium text-ink">Payment conversion.</strong> Verified confirmed bookings
				divided by all bookings created in the selected date range. No bookings shows a dash.
			</li>
			<li>
				<strong class="font-medium text-ink">Pending payment bookings.</strong> Booking status
				<em>pending_payment</em>, including bookings without a Virtual Account attempt.
			</li>
			<li>
				<strong class="font-medium text-ink">Most-booked vehicles.</strong> All statuses grouped by variant
				ID, using the saved vehicle label.
			</li>
		</ul>
	</details>
{/if}
