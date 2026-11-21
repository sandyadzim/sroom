<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Booking } from '#lib/types/booking.ts';
	import { bookingHistorySearch } from '#lib/utils/booking-history.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { onMount } from 'svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import BookingStatusBadge from './BookingStatusBadge.svelte';
	let {
		booking,
		search = ''
	}: { booking: Booking; search?: ReturnType<typeof bookingHistorySearch> } = $props();
	let now = $state(Date.now());
	const canContinuePayment = $derived(
		booking.status === 'pending_payment' && now < Date.parse(booking.hold_expires_at)
	);
	onMount(() => {
		const remaining = Date.parse(booking.hold_expires_at) - Date.now();
		if (booking.status !== 'pending_payment' || remaining <= 0) {
			now = Date.now();
			return;
		}
		const timer = setTimeout(() => {
			now = Date.now();
		}, remaining);
		return () => clearTimeout(timer);
	});
</script>

<article class="min-w-0 rounded-card border border-line bg-panel p-4 sm:p-5">
	<div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
		<h2 class="min-w-0 text-lg font-bold tracking-tight wrap-anywhere">{booking.vehicle_label}</h2>
		<BookingStatusBadge status={booking.status} />
	</div>
	<dl class="mt-4 grid grid-cols-1 gap-x-5 gap-y-3 text-sm sm:grid-cols-2">
		<div>
			<dt class="text-ink-muted">Pickup</dt>
			<dd class="mt-0.5 font-medium">
				<time datetime={booking.pickup_at}>{formatWitaDateTime(booking.pickup_at)} WITA</time>
			</dd>
		</div>
		<div>
			<dt class="text-ink-muted">Return</dt>
			<dd class="mt-0.5 font-medium">
				<time datetime={booking.return_at}>{formatWitaDateTime(booking.return_at)} WITA</time>
			</dd>
		</div>
	</dl>
	<div class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-line pt-3">
		<p class="font-semibold tabular-nums">{formatIDR(booking.total_idr)}</p>
		<div class="flex flex-wrap items-center gap-1">
			{#if canContinuePayment}<Button
					href={resolve('/payment/[bookingId]', { bookingId: booking.id })}
					variant="secondary">Continue Payment</Button
				>
			{:else if booking.status === 'confirmed'}<Button
					href={resolve(`/account/bookings/[bookingId]/invoice${search}`, {
						bookingId: booking.id
					})}
					variant="secondary">View Invoice</Button
				>{/if}
			<Button
				href={resolve(`/account/bookings/[bookingId]${search}`, { bookingId: booking.id })}
				variant="ghost"
				size="custom"
				class="min-h-11 rounded-lg px-3"
				aria-label={`View details for ${booking.vehicle_label}, booking ${booking.id}`}
				>View Details</Button
			>
		</div>
	</div>
</article>
