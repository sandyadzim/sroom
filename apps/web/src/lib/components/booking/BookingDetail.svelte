<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import type { Booking } from '#lib/types/booking.ts';
	import type { CustomerPayment } from '#lib/types/payment.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { rentalOutcome } from '#lib/utils/rental.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
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
	const labels = {
		pending_payment: 'Pending payment',
		confirmed: 'Confirmed',
		expired: 'Expired',
		payment_review: 'Under review'
	};
	const paymentLabels = {
		not_started: 'Not started',
		initiating: 'Preparing payment',
		pending: 'Pending',
		paid: 'Paid',
		expired: 'Expired',
		failed: 'Failed'
	};
	const outcome = $derived(rentalOutcome(booking, payment));
</script>

<svelte:head><title>Booking Detail — SROOM</title></svelte:head>
<Link
	href={resolve('/cars')}
	variant="plain"
	class="inline-flex min-h-11 items-center gap-2 text-sm text-ink-muted"
	><ArrowLeft size={16} aria-hidden="true" />Explore Cars</Link
>
<header class="mt-2 mb-8">
	<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">Booking Detail</h1>
	<div class="mt-3 flex flex-wrap items-center gap-3">
		<p class="text-sm text-ink-muted">{booking.vehicle_label}</p>
		<StatusBadge
			tone={booking.status === 'confirmed'
				? 'success'
				: booking.status === 'expired'
					? 'danger'
					: booking.status === 'payment_review'
						? 'info'
						: 'warning'}
			label={labels[booking.status]}
		/>
	</div>
</header>
<div class="grid items-start gap-6 xl:grid-cols-2">
	<div class="min-w-0 space-y-6">
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
				<div class="py-3">
					<dt class="text-ink-muted">Payment deadline</dt>
					<dd class="mt-1">
						<time datetime={booking.hold_expires_at}
							>{formatWitaDateTime(booking.hold_expires_at)} WITA</time
						>
					</dd>
				</div>
				{#if payment.payment_id}<div class="py-3">
						<dt class="text-ink-muted">Payment reference</dt>
						<dd class="mt-1 wrap-anywhere">{payment.payment_id}</dd>
					</div>{/if}
			</dl>
			<div class="mt-5 flex flex-wrap gap-3">
				{#if booking.status === 'confirmed'}<Button
						variant="secondary"
						href={resolve('/account/bookings/[bookingId]/invoice', { bookingId: booking.id })}
						>View Invoice</Button
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
			<ol class="mt-4 space-y-4 border-l border-line pl-4 text-sm">
				<li>
					<p class="font-medium">Reservation created</p>
					<time datetime={booking.created_at} class="mt-1 block text-ink-muted"
						>{formatWitaDateTime(booking.created_at)} WITA</time
					>
				</li>
				{#if payment.verified_at}<li>
						<p class="font-medium">Payment status verified</p>
						<time datetime={payment.verified_at} class="mt-1 block text-ink-muted"
							>{formatWitaDateTime(payment.verified_at)} WITA</time
						>
					</li>{/if}
				<li>
					<p class="font-medium">Current booking status</p>
					<p class="mt-1 text-ink-muted">{labels[booking.status]}</p>
				</li>
			</ol>
		</section>
	</div>
	<BookingSummary
		{booking}
		totalLabel={booking.status === 'confirmed' ? 'Total paid' : 'Booking total'}
	/>
</div>
