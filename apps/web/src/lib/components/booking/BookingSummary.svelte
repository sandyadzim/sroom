<script lang="ts">
	import { asset } from '$app/paths';
	import { checkoutPreviewImage, type BookingPreview } from '#lib/utils/booking-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	let {
		booking,
		bookingId,
		totalLabel = 'Total',
		pickupLocation,
		paymentReference,
		class: className
	}: {
		booking: BookingPreview;
		bookingId: string;
		totalLabel?: string;
		pickupLocation?: string;
		paymentReference?: string;
		class?: string;
	} = $props();
</script>

<aside
	id="booking-summary"
	tabindex="-1"
	aria-labelledby="booking-summary-heading"
	class={['min-w-0 overflow-hidden rounded-card border border-line bg-panel', className]}
>
	<img
		src={asset(checkoutPreviewImage)}
		alt={booking.carName}
		class="h-40 w-full bg-panel-muted object-contain p-2 lg:h-44"
	/>
	<div class="p-4 sm:p-6">
		<h2 id="booking-summary-heading" class="text-xl font-bold tracking-tight">Booking summary</h2>
		<p class="mt-1 text-xs font-medium break-all text-ink-muted">{bookingId.toUpperCase()}</p>
		<p class="mt-4 font-semibold">{booking.carName} · {booking.carTrim}</p>
		<dl class="mt-4 space-y-3 border-t border-line pt-4 text-sm">
			<div>
				<dt class="text-ink-muted">Pickup</dt>
				<dd>
					<time datetime={booking.pickupAt}>{formatWitaDateTime(booking.pickupAt)} WITA</time>
				</dd>
			</div>
			<div>
				<dt class="text-ink-muted">Return</dt>
				<dd>
					<time datetime={booking.returnAt}>{formatWitaDateTime(booking.returnAt)} WITA</time>
				</dd>
			</div>
			{#if pickupLocation}
				<div>
					<dt class="text-ink-muted">Pickup location</dt>
					<dd>{pickupLocation}</dd>
				</div>
			{/if}
			{#if paymentReference}
				<div>
					<dt class="text-ink-muted">Payment reference</dt>
					<dd class="wrap-anywhere">{paymentReference}</dd>
				</div>
			{/if}
			<div class="flex flex-wrap justify-between gap-2">
				<dt class="text-ink-muted">Daily rate</dt>
				<dd class="tabular-nums">{formatIDR(booking.dailyRate)}</dd>
			</div>
			<div class="flex flex-wrap justify-between gap-2">
				<dt class="text-ink-muted">Rental days</dt>
				<dd>{booking.days}</dd>
			</div>
			<div
				class="flex flex-wrap items-baseline justify-between gap-2 border-t border-line pt-4 font-bold"
			>
				<dt>{totalLabel}</dt>
				<dd class="text-xl tabular-nums">{formatIDR(booking.total)}</dd>
			</div>
		</dl>
	</div>
</aside>
