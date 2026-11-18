<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import DailyReportChart from '#lib/components/admin/DailyReportChart.svelte';
	import ReportMetrics from '#lib/components/admin/ReportMetrics.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import type { BookingStatus } from '#lib/types/admin-transactions.ts';
	import {
		getAdminReportPreview,
		getReportPeriod,
		isReportDate
	} from '#lib/utils/admin-report-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate } from '#lib/utils/date.ts';

	const range = $derived(getReportPeriod(page.url.searchParams));
	const report = $derived(getAdminReportPreview(range));
	const percent = new Intl.NumberFormat('en', { style: 'percent', maximumFractionDigits: 1 });
	const metrics = $derived([
		{ label: 'Bookings', value: report.total },
		{ label: 'Confirmed booking value', value: formatIDR(report.confirmedValue) },
		{
			label: 'Payment conversion',
			value: report.conversion === null ? '—' : percent.format(report.conversion)
		},
		{ label: 'Pending bookings', value: report.statusCounts.pending_payment },
		{ label: 'Expired bookings', value: report.statusCounts.expired }
	]);
	const statuses: BookingStatus[] = ['confirmed', 'pending_payment', 'payment_review', 'expired'];

	let startDate = $derived(range.from);
	let endDate = $derived(range.to);
	let startError = $state<string>();
	let endError = $state<string>();

	function applyDateRange(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		startError = isReportDate(startDate) ? undefined : 'Choose a start date.';
		endError = !isReportDate(endDate)
			? 'Choose an end date.'
			: startDate > endDate
				? 'End date must be on or after the start date.'
				: undefined;
		if (startError || endError) {
			event.currentTarget
				.querySelector<HTMLInputElement>(`[name=${startError ? 'from' : 'to'}]`)
				?.focus();
			return;
		}
		const query = new URLSearchParams({ from: startDate, to: endDate });
		void goto(resolve(`/admin/(staff)/reports?${query}`), { reset: false });
	}

	function resetDateRange() {
		startDate = range.from;
		endDate = range.to;
		startError = undefined;
		endError = undefined;
		void goto(resolve('/admin/(staff)/reports'), { reset: false });
	}
</script>

<svelte:head><title>Reports — SROOM</title></svelte:head>

<header class="mb-6">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Reports</h1>
	<p class="mt-2 text-sm leading-6 text-ink-muted">
		Simulated booking and payment figures by booking creation date in WITA.
	</p>
</header>

<form
	method="get"
	novalidate
	onsubmit={applyDateRange}
	class="mb-6 grid gap-4 rounded-xl border border-line bg-page p-4 sm:grid-cols-2 sm:p-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
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

<p role="status" class="mb-3 text-sm text-ink-muted">
	{formatWitaDate(range.from)} – {formatWitaDate(range.to)} · {report.total}
	{report.total === 1 ? 'booking' : 'bookings'}
</p>
<ReportMetrics items={metrics} />

<div class="mt-6 grid items-start gap-4 xl:grid-cols-2">
	<DailyReportChart
		title="Confirmed booking value by day"
		data={report.days}
		format={formatIDR}
		unit={250_000}
	/>
	<section aria-labelledby="booking-status-title" class="rounded-xl border border-line p-4 sm:p-5">
		<h2 id="booking-status-title" class="text-lg font-semibold tracking-tight">
			Bookings by status
		</h2>
		{#if report.total}
			<ul class="mt-4 space-y-3" role="list">
				{#each statuses as status (status)}
					<li>
						<div class="flex items-center justify-between gap-3">
							<TransactionStatus value={{ booking: status }} />
							<span class="text-sm font-medium tabular-nums">{report.statusCounts[status]}</span>
						</div>
						<div aria-hidden="true" class="mt-2 h-1.5 overflow-hidden rounded-full bg-panel-muted">
							<div
								class="h-full rounded-full bg-primary"
								style:width={`${(100 * report.statusCounts[status]) / report.total}%`}
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

<section aria-labelledby="top-models-title" class="mt-6 rounded-xl border border-line p-4 sm:p-5">
	<h2 id="top-models-title" class="text-lg font-semibold tracking-tight">Most-booked models</h2>
	{#if report.models.length}
		<table class="mt-3 w-full border-collapse text-left text-sm">
			<thead class="border-b border-line text-ink-muted">
				<tr
					><th scope="col" class="py-2 font-medium">Model</th><th
						scope="col"
						class="py-2 text-right font-medium">Bookings</th
					></tr
				>
			</thead>
			<tbody class="divide-y divide-line">
				{#each report.models as model (model.model)}
					<tr
						><th scope="row" class="py-2.5 font-medium">{model.model}</th><td
							class="py-2.5 text-right tabular-nums">{model.bookings}</td
						></tr
					>
				{/each}
			</tbody>
		</table>
	{:else}
		<p class="mt-3 text-sm text-ink-muted">No models to rank for this period.</p>
	{/if}
</section>

<details class="mt-5 rounded-lg border border-line px-4 py-2 text-sm">
	<summary class="cursor-pointer rounded-sm py-2 font-medium text-primary"
		>How metrics are counted</summary
	>
	<ul class="space-y-2 pb-3 text-ink-muted">
		<li>
			<strong class="font-medium text-ink">Confirmed booking value.</strong> Sum of confirmed bookings
			with a paid payment.
		</li>
		<li>
			<strong class="font-medium text-ink">Payment conversion.</strong> Confirmed, paid bookings divided
			by all bookings created in the selected date range.
		</li>
		<li>
			<strong class="font-medium text-ink">Most-booked models.</strong> Count of bookings for each model,
			across all statuses.
		</li>
	</ul>
</details>
