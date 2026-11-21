<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowRight, CalendarDays, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import BookingCard from '#lib/components/booking/BookingCard.svelte';
	import CustomerRequestState from '#lib/components/booking/CustomerRequestState.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { customerBookingsQuery } from '#lib/queries/rental.ts';
	import {
		bookingHistorySearch,
		bookingStatuses,
		bookingPageCount,
		readBookingFilters
	} from '#lib/utils/booking-history.ts';
	import type { BookingStatus } from '#lib/types/booking.ts';

	const profile = createQuery(() => customerProfileQuery);
	const selection = $derived(readBookingFilters(page.url.searchParams));
	const search = $derived(bookingHistorySearch(selection.status, selection.page));
	const bookings = createQuery(() => customerBookingsQuery(profile.data?.id ?? '', selection));
	const pageCount = $derived(bookingPageCount(bookings.data?.page.total ?? 0));
	const correctingPage = $derived(!!bookings.data && selection.page > pageCount);
	const filters = [
		{ value: 'all', label: 'All bookings' },
		...(Object.keys(bookingStatuses) as BookingStatus[]).map((value) => ({
			value,
			label: bookingStatuses[value].label
		}))
	] as const;

	$effect(() => {
		if (page.url.search !== search)
			void goto(resolve(`/account/bookings${search}`), { replace: true, reset: false });
	});
	$effect(() => {
		if (correctingPage && !bookings.isFetching && !bookings.isError) {
			void goto(resolve(`/account/bookings${bookingHistorySearch(selection.status, pageCount)}`), {
				replace: true,
				reset: false
			});
		}
	});
</script>

<svelte:head><title>My Bookings — SROOM</title></svelte:head>
<header class="mb-7 sm:mb-8">
	<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">My Bookings</h1>
	<p class="mt-2 text-sm leading-6 text-ink-muted sm:text-base">
		View your rental schedules and booking updates.
	</p>
</header>
<nav aria-label="Filter bookings" class="mb-5 flex flex-wrap gap-1 border-b border-line pb-4">
	{#each filters as filter (filter.value)}<Link
			href={resolve(`/account/bookings${bookingHistorySearch(filter.value)}`)}
			variant="plain"
			aria-current={selection.status === filter.value ? 'true' : undefined}
			class="inline-flex min-h-11 items-center rounded-lg px-3 text-sm font-medium text-ink-muted transition-colors hover:bg-panel-muted hover:text-ink aria-current:bg-primary aria-current:text-on-primary motion-reduce:transition-none"
			>{filter.label}</Link
		>{/each}
</nav>
{#if bookings.isError || !bookings.data}
	<CustomerRequestState
		error={bookings.error}
		loading={bookings.isPending}
		label="bookings"
		retry={() => void bookings.refetch()}
	/>
{:else if correctingPage}<p role="status" class="py-6 text-sm text-ink-muted">
		Opening the last available page…
	</p>
{:else if bookings.data.data.length === 0}
	<section class="rounded-card border border-line bg-panel-muted/45 px-5 py-9 sm:px-8">
		<CalendarDays size={27} aria-hidden="true" class="text-ink-muted" />
		{#if selection.status === 'all'}<h2 class="mt-3 text-xl font-bold">
				You haven’t booked a car yet
			</h2>
			<p class="mt-2 text-sm leading-6 text-ink-muted">
				Your reservations and booking updates will appear here.
			</p>
			<Button href={resolve('/cars')} class="mt-5"
				>Explore Cars <ArrowRight size={16} aria-hidden="true" /></Button
			>
		{:else}<h2 class="mt-3 text-xl font-bold">
				No {bookingStatuses[selection.status].label.toLowerCase()} bookings
			</h2>
			<p class="mt-2 text-sm text-ink-muted">Choose another filter to see other reservations.</p>
			<Button href={resolve('/account/bookings')} variant="secondary" class="mt-5"
				>Show all bookings</Button
			>{/if}
	</section>
{:else}
	<p role="status" aria-live="polite" class="mb-3 text-sm text-ink-muted">
		{bookings.data.page.total}
		{bookings.data.page.total === 1 ? 'booking' : 'bookings'}
	</p>
	<ul class="space-y-3" aria-label="Your bookings">
		{#each bookings.data.data as booking (booking.id)}<li>
				<BookingCard {booking} {search} />
			</li>{/each}
	</ul>
	{#if pageCount > 1}
		<nav aria-label="Booking pagination" class="mt-5 flex items-center justify-end gap-3">
			{#if selection.page > 1}<Button
					href={resolve(
						`/account/bookings${bookingHistorySearch(selection.status, selection.page - 1)}`
					)}
					variant="secondary"
					size="icon"
					aria-label="Previous page"><ChevronLeft size={18} aria-hidden="true" /></Button
				>{:else}<Button variant="secondary" size="icon" aria-label="Previous page" disabled
					><ChevronLeft size={18} aria-hidden="true" /></Button
				>{/if}
			<p class="text-sm text-ink-muted tabular-nums">Page {selection.page} of {pageCount}</p>
			{#if selection.page < pageCount}<Button
					href={resolve(
						`/account/bookings${bookingHistorySearch(selection.status, selection.page + 1)}`
					)}
					variant="secondary"
					size="icon"
					aria-label="Next page"><ChevronRight size={18} aria-hidden="true" /></Button
				>{:else}<Button variant="secondary" size="icon" aria-label="Next page" disabled
					><ChevronRight size={18} aria-hidden="true" /></Button
				>{/if}
		</nav>
	{/if}
{/if}
