<script lang="ts">
	import { asset, resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowRight, CalendarDays, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import BookingStatusBadge from '#lib/components/booking/BookingStatusBadge.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import {
		bookingHistorySearch,
		bookingStatuses,
		readBookingFilters,
		type BookingStatus
	} from '#lib/utils/booking-history.ts';
	import {
		addBookingHistorySamples,
		checkoutPreviewImage,
		checkoutPreviewSlug,
		listBookingPreviews
	} from '#lib/utils/booking-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDate } from '#lib/utils/date.ts';

	const filters = [
		{ value: 'all', label: 'All bookings' },
		...(Object.keys(bookingStatuses) as BookingStatus[]).map((value) => ({
			value,
			label: bookingStatuses[value].label
		}))
	] as const;
	const pageSize = 5;
	let bookings = $state<ReturnType<typeof listBookingPreviews>>();
	let unavailable = $state(false);
	const selection = $derived(readBookingFilters(page.url.searchParams));
	const filtered = $derived(
		(bookings ?? []).filter(
			(booking) => selection.status === 'all' || (booking.outcome ?? 'pending') === selection.status
		)
	);
	const pageCount = $derived(Math.max(1, Math.ceil(filtered.length / pageSize)));
	const currentPage = $derived(Math.min(selection.page, pageCount));
	const visible = $derived(filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize));

	function loadBookings() {
		try {
			bookings = listBookingPreviews();
			unavailable = false;
		} catch {
			unavailable = true;
		}
	}

	onMount(() => {
		const scenario = page.url.searchParams.get('scenario');
		if (scenario === 'error') {
			unavailable = true;
			return;
		}
		try {
			if (scenario === 'sample') addBookingHistorySamples();
			loadBookings();
		} catch {
			unavailable = true;
		}
	});
</script>

<svelte:head><title>My Bookings — SROOM</title></svelte:head>

<header class="mb-7 sm:mb-8">
	<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">My Bookings</h1>
	<p class="mt-2 text-sm leading-6 text-ink-muted sm:text-base">
		View your rental schedules and payment status.
	</p>
</header>

{#if unavailable}
	<section class="rounded-card border border-line p-5 sm:p-7">
		<h2 class="text-xl font-bold">Couldn’t load your bookings</h2>
		<p role="alert" class="mt-2 text-sm text-ink-muted">Try loading your booking history again.</p>
		<Button variant="secondary" class="mt-4" onclick={loadBookings}>Try again</Button>
	</section>
{:else if bookings === undefined}
	<p role="status" class="text-sm text-ink-muted">Loading your bookings…</p>
{:else}
	{#if bookings.length > 0}
		<nav aria-label="Filter bookings" class="mb-5 flex flex-wrap gap-1 border-b border-line pb-4">
			{#each filters as filter (filter.value)}
				<Link
					href={resolve(`/account/bookings${bookingHistorySearch(filter.value)}`)}
					variant="plain"
					aria-current={selection.status === filter.value ? 'true' : undefined}
					class="inline-flex min-h-11 items-center rounded-lg px-3 text-sm font-medium text-ink-muted transition-colors hover:bg-panel-muted hover:text-ink aria-current:bg-primary aria-current:text-on-primary motion-reduce:transition-none"
				>
					{filter.label}
				</Link>
			{/each}
		</nav>
	{/if}

	{#if visible.length === 0}
		<section class="rounded-card border border-line bg-panel-muted/45 px-5 py-9 sm:px-8">
			<CalendarDays size={27} aria-hidden="true" class="text-ink-muted" />
			{#if bookings.length === 0}
				<h2 class="mt-3 text-xl font-bold">You haven’t booked a car yet</h2>
				<p class="mt-2 text-sm leading-6 text-ink-muted">
					Your reservations and payment updates will appear here.
				</p>
				<div class="mt-5 flex flex-wrap gap-2">
					<Button href={resolve('/cars')}
						>Explore Cars <ArrowRight size={16} aria-hidden="true" /></Button
					>
					<Button
						href={resolve('/checkout/[slug]', { slug: checkoutPreviewSlug })}
						variant="secondary">Open Checkout</Button
					>
				</div>
			{:else}
				<h2 class="mt-3 text-xl font-bold">
					No {selection.status === 'all'
						? ''
						: bookingStatuses[selection.status].label.toLowerCase()} bookings
				</h2>
				<p class="mt-2 text-sm text-ink-muted">Choose another filter to see other reservations.</p>
				<Button href={resolve('/account/bookings')} variant="secondary" class="mt-5"
					>Show all bookings</Button
				>
			{/if}
		</section>
	{:else}
		<p role="status" class="mb-3 text-sm text-ink-muted">
			{filtered.length}
			{filtered.length === 1 ? 'booking' : 'bookings'}
		</p>
		<ul class="space-y-3">
			{#each visible as booking (booking.id)}
				<li>
					<article
						class="grid gap-4 rounded-card border border-line bg-panel p-4 sm:grid-cols-[7.5rem_minmax(0,1fr)] sm:p-5"
					>
						<div class="hidden h-24 rounded-xl bg-panel-muted sm:grid sm:place-items-center">
							<img
								src={asset(checkoutPreviewImage)}
								alt=""
								class="max-h-full w-full object-contain p-1.5"
							/>
						</div>
						<div class="min-w-0">
							<div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
								<h2 class="text-lg font-bold tracking-tight">
									{booking.carName}
									<span class="text-sm font-medium text-ink-muted">· {booking.carTrim}</span>
								</h2>
								<BookingStatusBadge status={booking.outcome ?? 'pending'} />
							</div>
							<p class="mt-1 text-xs wrap-anywhere text-ink-muted">{booking.id.toUpperCase()}</p>
							<dl class="mt-4 grid grid-cols-2 gap-x-5 gap-y-2 text-sm">
								<div>
									<dt class="text-ink-muted">Pickup</dt>
									<dd class="mt-0.5 font-medium">
										<time datetime={booking.pickupAt}>{formatWitaDate(booking.pickupAt)}</time>
									</dd>
								</div>
								<div>
									<dt class="text-ink-muted">Return</dt>
									<dd class="mt-0.5 font-medium">
										<time datetime={booking.returnAt}>{formatWitaDate(booking.returnAt)}</time>
									</dd>
								</div>
							</dl>
							<div
								class="mt-4 flex flex-wrap items-center justify-between gap-2 border-t border-line pt-3"
							>
								<p class="font-semibold tabular-nums">{formatIDR(booking.total)}</p>
								<Button
									href={resolve(
										`/account/bookings/[bookingId]${bookingHistorySearch(selection.status, currentPage)}`,
										{ bookingId: booking.id }
									)}
									variant="ghost"
									size="custom"
									class="rounded-lg px-3"
									aria-label={`View details for ${booking.carName}, booking ${booking.id.toUpperCase()}`}
								>
									View Details <ArrowRight size={16} aria-hidden="true" />
								</Button>
							</div>
						</div>
					</article>
				</li>
			{/each}
		</ul>
		{#if pageCount > 1}
			<nav aria-label="Booking pagination" class="mt-5 flex items-center justify-end gap-3">
				{#if currentPage > 1}
					<Button
						href={resolve(
							`/account/bookings${bookingHistorySearch(selection.status, currentPage - 1)}`
						)}
						variant="secondary"
						size="icon"
						aria-label="Previous page"><ChevronLeft size={18} aria-hidden="true" /></Button
					>
				{:else}
					<Button variant="secondary" size="icon" aria-label="Previous page" disabled
						><ChevronLeft size={18} aria-hidden="true" /></Button
					>
				{/if}
				<p class="text-sm text-ink-muted tabular-nums">Page {currentPage} of {pageCount}</p>
				{#if currentPage < pageCount}
					<Button
						href={resolve(
							`/account/bookings${bookingHistorySearch(selection.status, currentPage + 1)}`
						)}
						variant="secondary"
						size="icon"
						aria-label="Next page"><ChevronRight size={18} aria-hidden="true" /></Button
					>
				{:else}
					<Button variant="secondary" size="icon" aria-label="Next page" disabled
						><ChevronRight size={18} aria-hidden="true" /></Button
					>
				{/if}
			</nav>
		{/if}
	{/if}
{/if}
