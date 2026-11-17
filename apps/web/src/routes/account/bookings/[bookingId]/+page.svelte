<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft, ArrowRight } from '@lucide/svelte';
	import BookingStatusBadge from '#lib/components/booking/BookingStatusBadge.svelte';
	import BookingSummary from '#lib/components/booking/BookingSummary.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import {
		bookingHistorySearch,
		bookingStatuses,
		readBookingFilters
	} from '#lib/utils/booking-history.ts';
	import {
		bookingPreviewPaymentReference,
		checkoutPreviewSlug,
		readBookingPreview
	} from '#lib/utils/booking-preview.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let booking = $state<ReturnType<typeof readBookingPreview>>();
	let bookingExpiredByClock = $state(false);
	const selection = $derived(readBookingFilters(page.url.searchParams));
	const backHref = $derived(
		resolve(`/account/bookings${bookingHistorySearch(selection.status, selection.page)}`)
	);
	const invoiceHref = $derived(
		resolve(
			`/account/bookings/[bookingId]/invoice${bookingHistorySearch(selection.status, selection.page)}`,
			{
				bookingId: data.bookingId
			}
		)
	);
	const paymentHref = $derived(resolve('/payment/[bookingId]', { bookingId: data.bookingId }));
	const status = $derived(booking?.outcome ?? 'pending');
	const paymentStatus = $derived(
		{
			pending: bookingExpiredByClock ? 'Awaiting status update' : 'Awaiting payment',
			confirmed: 'Paid',
			expired: 'Payment not confirmed',
			review: 'Under review'
		}[status]
	);

	$effect(() => {
		const saved = readBookingPreview(data.bookingId);
		booking = saved;
		const timeLeft = saved ? saved.expiresAt - Date.now() : 0;
		bookingExpiredByClock = timeLeft <= 0;
		if (saved && !saved.outcome && timeLeft > 0) {
			const timer = setTimeout(() => (bookingExpiredByClock = true), timeLeft);
			return () => clearTimeout(timer);
		}
	});
</script>

<svelte:head><title>Booking Detail — SROOM</title></svelte:head>

<Link
	href={backHref}
	variant="plain"
	class="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} aria-hidden="true" />
	My Bookings
</Link>

{#if booking === undefined}
	<p role="status" class="mt-4 text-sm text-ink-muted">Loading booking details…</p>
{:else if !booking}
	<section class="mt-4 rounded-card border border-line p-5 sm:p-7">
		<h1 class="text-2xl font-bold tracking-tight">Booking unavailable</h1>
		<p class="mt-2 text-sm leading-6 text-ink-muted">
			This booking couldn’t be found. Return to My Bookings to view your reservations.
		</p>
	</section>
{:else}
	<header class="mt-2 mb-7 sm:mb-8">
		<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">Booking Detail</h1>
		<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2">
			<p class="text-sm text-ink-muted">{booking.carName} · {booking.carTrim}</p>
			<BookingStatusBadge {status} />
		</div>
	</header>

	<div
		class="grid items-start gap-6 xl:grid-cols-[minmax(0,1.06fr)_minmax(0,1fr)] xl:grid-rows-[auto_1fr]"
	>
		<div class="min-w-0">
			<section aria-labelledby="payment-heading" class="rounded-card border border-line p-4 sm:p-5">
				<h2 id="payment-heading" class="text-xl font-bold">Payment</h2>
				<dl class="mt-4 divide-y divide-line border-y border-line text-sm">
					<div class="flex flex-wrap justify-between gap-2 py-3">
						<dt class="text-ink-muted">Status</dt>
						<dd class="font-medium">{paymentStatus}</dd>
					</div>
					<div class="flex flex-wrap justify-between gap-2 py-3">
						<dt class="text-ink-muted">Method</dt>
						<dd class="font-medium">Virtual Account</dd>
					</div>
					{#if status === 'pending'}
						<div class="py-3">
							<dt class="text-ink-muted">Payment deadline</dt>
							<dd class="mt-0.5 font-medium">
								<time datetime={new Date(booking.expiresAt).toISOString()}
									>{formatWitaDateTime(new Date(booking.expiresAt).toISOString())} WITA</time
								>
							</dd>
						</div>
					{/if}
				</dl>
				<div class="mt-4">
					{#if status === 'pending'}
						{#if bookingExpiredByClock}<p class="mb-3 text-sm leading-6 text-ink-muted">
								The deadline has passed. Check the latest status before paying.
							</p>{/if}
						<Button href={paymentHref}
							>{bookingExpiredByClock ? 'Check Payment Status' : 'Continue Payment'}
							<ArrowRight size={16} aria-hidden="true" /></Button
						>
					{:else if status === 'confirmed'}
						<Button href={invoiceHref} variant="secondary">View Invoice</Button>
					{:else if status === 'expired'}
						<p class="mb-3 text-sm leading-6 text-ink-muted">
							This reservation has ended. Choose a new schedule to check availability again.
						</p>
						<Button href={resolve('/checkout/[slug]', { slug: checkoutPreviewSlug })}
							>Choose a New Schedule <ArrowRight size={16} aria-hidden="true" /></Button
						>
					{:else}
						<p class="mb-3 text-sm leading-6 text-ink-muted">
							Keep your payment reference and wait for an update before paying again.
						</p>
						<Button
							href={resolve('/bookings/[bookingId]/[outcome]', {
								bookingId: data.bookingId,
								outcome: 'review'
							})}>View Review Status <ArrowRight size={16} aria-hidden="true" /></Button
						>
					{/if}
				</div>
			</section>
		</div>
		<BookingSummary
			{booking}
			bookingId={data.bookingId}
			pickupLocation="Samarinda, East Kalimantan"
			paymentReference={status === 'confirmed' || status === 'review'
				? bookingPreviewPaymentReference(data.bookingId)
				: undefined}
			totalLabel={status === 'confirmed' ? 'Total paid' : 'Booking total'}
			class="xl:col-start-2 xl:row-span-2 xl:row-start-1"
		/>
		<section aria-labelledby="history-heading" class="rounded-card border border-line p-4 sm:p-5">
			<h2 id="history-heading" class="text-xl font-bold">Booking timeline</h2>
			<ol class="mt-4 space-y-4 border-l border-line pl-4 text-sm">
				<li
					class="relative before:absolute before:top-[0.55rem] before:-left-[1.27rem] before:size-2 before:rounded-full before:bg-ink-muted"
				>
					<p class="font-medium">Reservation created</p>
					{#if booking.createdAt}<p class="mt-1 text-ink-muted">
							{formatWitaDateTime(new Date(booking.createdAt).toISOString())} WITA
						</p>{/if}
				</li>
				{#if status !== 'pending'}
					<li
						class="relative before:absolute before:top-[0.55rem] before:-left-[1.27rem] before:size-2 before:rounded-full before:bg-primary"
					>
						<p class="font-medium">{bookingStatuses[status].label}</p>
						{#if booking.outcomeAt}<p class="mt-1 text-ink-muted">
								{formatWitaDateTime(new Date(booking.outcomeAt).toISOString())} WITA
							</p>{/if}
					</li>
				{/if}
			</ol>
		</section>
	</div>
{/if}
