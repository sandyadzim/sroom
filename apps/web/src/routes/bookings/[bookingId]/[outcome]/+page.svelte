<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { ArrowRight, CircleCheck, Clock3, RefreshCw, ShieldAlert } from '@lucide/svelte';
	import BookingSummary from '#lib/components/booking/BookingSummary.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import {
		bookingOutcomeRoute,
		checkoutPreviewSlug,
		readBookingPreview
	} from '#lib/utils/booking-preview.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const content = {
		success: {
			title: 'Booking confirmed',
			icon: CircleCheck,
			tone: 'bg-positive/10 text-positive',
			description: 'Your payment is complete and your car is reserved for the dates shown.'
		},
		expired: {
			title: 'Reservation expired',
			icon: Clock3,
			tone: 'bg-caution/10 text-caution',
			description: 'Payment was not confirmed within 15 minutes. This reservation has ended.'
		},
		review: {
			title: 'Payment under review',
			icon: ShieldAlert,
			tone: 'bg-panel-muted text-primary',
			description:
				'We’re checking your payment and reservation. Your booking is still awaiting confirmation.'
		}
	};

	const checkoutHref = $derived(resolve('/checkout/[slug]', { slug: checkoutPreviewSlug }));
	const paymentHref = $derived(resolve('/payment/[bookingId]', { bookingId: data.bookingId }));
	let booking = $state<ReturnType<typeof readBookingPreview>>();
	let reviewMessage = $state('');
	const view = $derived(
		booking?.outcome ? content[bookingOutcomeRoute(booking.outcome)] : undefined
	);
	const paymentReference = $derived(`VA-${data.bookingId.slice(-12).toUpperCase()}`);

	$effect(() => {
		const saved = readBookingPreview(data.bookingId);
		if (saved?.outcome && bookingOutcomeRoute(saved.outcome) !== data.outcome) {
			void goto(
				resolve('/bookings/[bookingId]/[outcome]', {
					bookingId: data.bookingId,
					outcome: bookingOutcomeRoute(saved.outcome)
				}),
				{ replace: true }
			).catch(() => (booking = null));
			return;
		}
		booking = saved;
	});

	function checkReview() {
		reviewMessage =
			readBookingPreview(data.bookingId)?.outcome === 'review'
				? 'Your payment is still under review.'
				: 'Could not check your payment status. Please try again.';
	}
</script>

<svelte:head>
	<title>{view?.title ?? 'Booking status'} — SROOM</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="page-container py-8 sm:py-12 lg:py-16">
	{#if booking === undefined}
		<p role="status" class="text-sm text-ink-muted">Loading booking status…</p>
	{:else if !booking || !view}
		<section class="max-w-xl rounded-card border border-line p-5 sm:p-8">
			<h1 class="text-3xl font-bold tracking-tight">Booking status unavailable</h1>
			<p class="mt-3 text-sm leading-6 text-ink-muted">
				{booking
					? 'Check Payment for the latest status of this booking.'
					: 'Open Checkout in this tab to start a reservation.'}
			</p>
			<Button href={booking ? paymentHref : checkoutHref} class="mt-5">
				{booking ? 'Check Payment Status' : 'Go to checkout'}
			</Button>
		</section>
	{:else}
		{@const Icon = view.icon}
		<div
			class="mx-auto grid max-w-6xl items-start gap-8 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:gap-16"
		>
			<section aria-labelledby="outcome-heading" class="min-w-0 lg:pt-5">
				<div class={['grid size-14 place-items-center rounded-2xl sm:size-16', view.tone]}>
					<Icon size={28} strokeWidth={1.8} aria-hidden="true" />
				</div>
				<h1 id="outcome-heading" class="mt-5 text-4xl font-extrabold tracking-tight sm:text-5xl">
					{view.title}
				</h1>
				<p class="mt-3 max-w-lg text-base leading-7 text-ink-muted">{view.description}</p>

				<div class="mt-7 border-t border-line pt-6 sm:mt-8">
					{#if booking.outcome === 'confirmed'}
						<p class="max-w-lg text-sm leading-6 text-ink-muted">
							A confirmation email will be sent separately.
						</p>
						<div class="mt-4 flex flex-wrap items-center gap-2">
							<Button href="#booking-summary"
								>View Booking <ArrowRight size={17} aria-hidden="true" /></Button
							>
							<Button variant="secondary" disabled aria-describedby="invoice-note"
								>View Invoice</Button
							>
						</div>
						<p id="invoice-note" class="mt-2 text-sm text-ink-muted">
							Invoice access is coming soon.
						</p>
					{:else if booking.outcome === 'expired'}
						<p class="max-w-lg text-sm leading-6 text-ink-muted">
							Choose a new schedule to check availability again. Do not pay using the previous
							Virtual Account.
						</p>
						<Button href={checkoutHref} class="mt-4"
							>Choose a New Schedule <ArrowRight size={17} aria-hidden="true" /></Button
						>
					{:else}
						<p class="max-w-lg text-sm leading-6 text-ink-muted">
							Keep your payment reference and wait for a status update before making any further
							payment.
						</p>
						<div class="mt-4 flex flex-wrap items-center gap-2">
							<Button onclick={checkReview}
								><RefreshCw size={16} aria-hidden="true" /> Check Payment Status</Button
							>
							<Button href="#booking-summary" variant="secondary">View Booking</Button>
						</div>
						<p role="status" class="mt-2 text-sm text-ink-muted">{reviewMessage}</p>
					{/if}
				</div>
			</section>

			<BookingSummary
				{booking}
				bookingId={data.bookingId}
				pickupLocation={booking.outcome === 'confirmed' ? 'Samarinda, East Kalimantan' : undefined}
				paymentReference={booking.outcome !== 'expired' ? paymentReference : undefined}
				totalLabel={booking.outcome === 'confirmed' ? 'Total paid' : 'Booking total'}
			/>
		</div>
	{/if}
</div>
