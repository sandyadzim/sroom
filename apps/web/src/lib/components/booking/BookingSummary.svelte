<script lang="ts">
	import type { Booking } from '#lib/types/booking.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	let {
		booking,
		bookingId = booking.id,
		totalLabel = 'Booking total',
		paymentReference,
		class: className
	}: {
		booking: Pick<
			Booking,
			| 'id'
			| 'vehicle_label'
			| 'pickup_at'
			| 'return_at'
			| 'rental_days'
			| 'daily_rate_idr'
			| 'total_idr'
		>;
		bookingId?: string;
		totalLabel?: string;
		paymentReference?: string;
		class?: string;
	} = $props();
</script>

<aside
	id="booking-summary"
	tabindex="-1"
	aria-labelledby="booking-summary-heading"
	class={['min-w-0 rounded-card border border-line bg-panel p-4 sm:p-6', className]}
>
	<h2 id="booking-summary-heading" class="text-xl font-bold tracking-tight">Booking summary</h2>
	<p class="mt-2 text-xs font-medium wrap-anywhere text-ink-muted">{bookingId}</p>
	<p class="mt-5 font-semibold">{booking.vehicle_label}</p>
	<dl class="mt-4 space-y-4 border-t border-line pt-4 text-sm">
		<div>
			<dt class="text-ink-muted">Pickup</dt>
			<dd>
				<time datetime={booking.pickup_at}>{formatWitaDateTime(booking.pickup_at)} WITA</time>
			</dd>
		</div>
		<div>
			<dt class="text-ink-muted">Return</dt>
			<dd>
				<time datetime={booking.return_at}>{formatWitaDateTime(booking.return_at)} WITA</time>
			</dd>
		</div>
		{#if paymentReference}<div>
				<dt class="text-ink-muted">Payment reference</dt>
				<dd class="wrap-anywhere">{paymentReference}</dd>
			</div>{/if}
		<div class="flex flex-wrap justify-between gap-2">
			<dt class="text-ink-muted">Daily rate</dt>
			<dd class="tabular-nums">{formatIDR(booking.daily_rate_idr)}</dd>
		</div>
		<div class="flex flex-wrap justify-between gap-2">
			<dt class="text-ink-muted">Rental days</dt>
			<dd>{booking.rental_days}</dd>
		</div>
		<div
			class="flex flex-wrap items-baseline justify-between gap-2 border-t border-line pt-4 font-bold"
		>
			<dt>{totalLabel}</dt>
			<dd class="text-xl tabular-nums">{formatIDR(booking.total_idr)}</dd>
		</div>
	</dl>
</aside>
