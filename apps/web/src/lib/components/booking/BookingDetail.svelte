<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import {
		bookingHistorySearch,
		bookingStatuses,
		readBookingFilters
	} from '#lib/utils/booking-history.ts';
	import { ArrowLeft } from '@lucide/svelte';
	import type { Booking } from '#lib/types/booking.ts';
	import type { CustomerPayment } from '#lib/types/payment.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { rentalOutcome } from '#lib/utils/rental.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import BookingStatusBadge from './BookingStatusBadge.svelte';
	import BookingSummary from './BookingSummary.svelte';
	let {
		booking,
		payment,
		checkPayment,
		checking
	}: {
		booking: Booking;
		payment: CustomerPayment;
		checkPayment: () => Promise<void>;
		checking: boolean;
	} = $props();
	const selection = $derived(readBookingFilters(page.url.searchParams));
	const search = $derived(bookingHistorySearch(selection.status, selection.page));
	const paymentLabels = {
		not_started: 'Not started',
		initiating: 'Preparing payment',
		pending: 'Pending',
		paid: 'Paid',
		expired: 'Expired',
		failed: 'Failed'
	};
	const outcome = $derived(rentalOutcome(booking, payment));
	const timeline = $derived([
		{ title: 'Reservation created', time: booking.created_at, description: null },
		...(payment.verified_at
			? [{ title: 'Payment status verified', time: payment.verified_at, description: null }]
			: []),
		{
			title: 'Current booking status',
			time: null,
			description: bookingStatuses[booking.status].label
		}
	]);
</script>

<svelte:head><title>Booking Detail — SROOM</title></svelte:head>
<Link
	href={resolve(`/account/bookings${search}`)}
	variant="plain"
	class="inline-flex min-h-11 items-center gap-2 text-sm text-ink-muted"
	><ArrowLeft size={16} aria-hidden="true" />Back to My Bookings</Link
>
<header class="mt-2 mb-8">
	<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">Booking Detail</h1>
	<div class="mt-3 flex flex-wrap items-center gap-3">
		<p class="text-sm text-ink-muted">{booking.vehicle_label}</p>
		<BookingStatusBadge status={booking.status} />
	</div>
</header>
<div class="grid grid-cols-1 gap-6">
	<BookingSummary
		{booking}
		totalLabel={booking.status === 'confirmed' ? 'Total paid' : 'Booking total'}
	/>
	<section aria-labelledby="payment-title" class="rounded-card border border-line p-4 sm:p-5">
		<h2 id="payment-title" class="text-xl font-bold">Payment</h2>
		<dl class="mt-4 divide-y divide-line border-y border-line text-sm">
			<div class="flex flex-wrap justify-between gap-2 py-3">
				<dt class="text-ink-muted">Status</dt>
				<dd class="font-medium">
					{payment.review_required ? 'Under review' : paymentLabels[payment.status]}
				</dd>
			</div>
			{#if payment.bank}<div class="flex flex-wrap justify-between gap-2 py-3">
					<dt class="text-ink-muted">Bank</dt>
					<dd class="font-medium">BNI</dd>
				</div>{/if}
			<div class="flex min-w-0 flex-col gap-1 py-3 sm:flex-row sm:justify-between sm:gap-6">
				<dt class="text-ink-muted">Payment deadline</dt>
				<dd class="min-w-0 sm:text-right">
					<time datetime={booking.hold_expires_at}
						>{formatWitaDateTime(booking.hold_expires_at)} WITA</time
					>
				</dd>
			</div>
			{#if payment.payment_id}<div
					class="flex min-w-0 flex-col gap-1 py-3 sm:flex-row sm:justify-between sm:gap-6"
				>
					<dt class="text-ink-muted">Payment reference</dt>
					<dd class="min-w-0 wrap-anywhere sm:text-right">{payment.payment_id}</dd>
				</div>{/if}
		</dl>
		<div class="mt-5 flex flex-wrap gap-3">
			{#if booking.status === 'confirmed'}<Button
					variant="secondary"
					href={resolve(`/account/bookings/[bookingId]/invoice${search}`, {
						bookingId: booking.id
					})}>View Invoice</Button
				>
			{:else if outcome === 'expired'}<Button href={resolve('/cars')}>Explore Cars</Button>
			{:else if outcome === 'review'}<Button
					href={resolve('/bookings/[bookingId]/[outcome]', {
						bookingId: booking.id,
						outcome: 'review'
					})}>View Review Status</Button
				>
			{:else}<Button href={resolve('/payment/[bookingId]', { bookingId: booking.id })}
					>{Date.now() >= Date.parse(booking.hold_expires_at) || payment.status === 'paid'
						? 'View Payment Status'
						: 'Continue Payment'}</Button
				>{/if}
			<Button
				variant="ghost"
				disabled={checking}
				aria-busy={checking}
				onclick={() => void checkPayment()}>Check payment status</Button
			>
		</div>
	</section>
	<section aria-labelledby="timeline-title" class="rounded-card border border-line p-4 sm:p-5">
		<h2 id="timeline-title" class="text-xl font-bold">Booking timeline</h2>
		<ol class="mt-5 text-sm leading-6">
			{#each timeline as event, index (event.title)}
				<li
					aria-current={index === timeline.length - 1 ? 'step' : undefined}
					class="relative pb-6 pl-7 before:absolute before:top-3 before:-bottom-3 before:left-1.5 before:w-px before:bg-line last:pb-0 last:before:hidden"
				>
					<span
						aria-hidden="true"
						class={[
							'absolute top-1.5 left-0 size-3 rounded-full border-2 border-primary',
							index === timeline.length - 1 ? 'bg-primary ring-4 ring-primary/10' : 'bg-panel'
						]}
					></span>
					<p class="font-medium">{event.title}</p>
					{#if event.time}<time datetime={event.time} class="mt-1 block text-ink-muted tabular-nums"
							>{formatWitaDateTime(event.time)} WITA</time
						>{/if}
					{#if event.description}<p class="mt-1 text-ink-muted">{event.description}</p>{/if}
				</li>
			{/each}
		</ol>
	</section>
</div>
