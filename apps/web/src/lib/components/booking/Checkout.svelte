<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onDestroy, onMount } from 'svelte';
	import { isAxiosError } from 'axios';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ArrowLeft, ArrowRight, CalendarDays, CheckCircle2, MapPin } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import CustomerRequestState from './CustomerRequestState.svelte';
	import { fetchRentalQuote, createBooking } from '#lib/api/booking.ts';
	import { getApiError } from '#lib/api/error.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import { holdCommand, readPendingHold, clearPendingHold } from '#lib/utils/rental.ts';
	import {
		earliestPickupDate,
		validateSchedule,
		type Schedule
	} from '../../../routes/checkout/[slug]/schedule.ts';
	import type { CatalogCarDetail } from '#lib/types/catalog.ts';
	import type { CustomerSessionProfile } from '#lib/types/identity.ts';
	import type { HoldInput, PendingHold, QuoteInput, RentalQuote } from '#lib/types/booking.ts';
	let { car, customer }: { car: CatalogCarDetail; customer: CustomerSessionProfile } = $props();
	const toast = useToast();
	const client = useQueryClient();
	const check = createMutation(() => ({
		mutationFn: (input: QuoteInput) => fetchRentalQuote(input),
		retry: false
	}));
	const hold = createMutation(() => ({
		mutationFn: ({ payload, key }: PendingHold) => createBooking(payload, key),
		retry: false
	}));
	let ready = $state(false);
	let earliestPickup = $state(earliestPickupDate());
	let schedule = $state<Schedule>({
		pickupDate: '',
		pickupTime: '',
		returnDate: '',
		returnTime: ''
	});
	let touched = $state<Partial<Record<keyof Schedule, boolean>>>({});
	let checked = $state(false);
	let accepted = $state(false);
	let quote = $state<RentalQuote>();
	let pending = $state<PendingHold>();
	let feedback = $state<{ tone: 'success' | 'error'; message: string }>();
	let version = 0;
	let active = true;
	onDestroy(() => {
		active = false;
		version += 1;
	});
	const sessionError = $derived(
		[hold.error, check.error].find((cause) => isAxiosError(cause) && cause.response?.status === 401)
	);
	const carName = $derived(`${car.brand_name} ${car.model}`);
	const checking = $derived(check.isPending);
	const reserving = $derived(hold.isPending);
	const errors = $derived(validateSchedule(schedule, earliestPickup));
	const valid = $derived(Object.values(errors).every((message) => !message));
	const days = $derived(quote?.rental_days ?? null);
	const total = $derived(quote?.total_idr ?? null);
	const canReserve = $derived(
		ready &&
			car.status === 'active' &&
			accepted &&
			!checking &&
			!reserving &&
			(!!pending || (valid && quote?.available === true))
	);
	onMount(() => {
		ready = true;
		const saved = readPendingHold(customer.id, car.slug);
		if (saved?.payload.variant_id === car.id) {
			pending = saved;
			schedule = {
				pickupDate: saved.payload.pickup_at.slice(0, 10),
				pickupTime: saved.payload.pickup_at.slice(11, 16),
				returnDate: saved.payload.return_at.slice(0, 10),
				returnTime: saved.payload.return_at.slice(11, 16)
			};
		}
	});
	function scheduleChanged() {
		version += 1;
		quote = undefined;
		pending = undefined;
		feedback = undefined;
	}
	async function checkAvailability(form: HTMLFormElement | null) {
		earliestPickup = earliestPickupDate();
		checked = true;
		const invalid = (Object.keys(errors) as (keyof Schedule)[]).find((field) => errors[field]);
		if (invalid) {
			form?.querySelector<HTMLInputElement>(`[name="${invalid}"]`)?.focus();
			return;
		}
		if (checking || reserving) return;
		const current = ++version;
		quote = undefined;
		pending = undefined;
		feedback = undefined;
		try {
			const result = await check.mutateAsync({
				variant_id: car.id,
				pickup_at: `${schedule.pickupDate}T${schedule.pickupTime}:00+08:00`,
				return_at: `${schedule.returnDate}T${schedule.returnTime}:00+08:00`
			});
			if (version !== current) return;
			quote = result;
			feedback = result.available
				? { tone: 'success', message: 'Available for your selected schedule.' }
				: {
						tone: 'error',
						message: 'No cars available for this schedule. Try different dates or times.'
					};
		} catch (cause) {
			if (version !== current) return;
			const message =
				getApiError(cause)?.message ?? 'Could not check availability. Please try again.';
			feedback = { tone: 'error', message };
			toast({ kind: 'error', message });
		}
	}
	async function reserve(event: SubmitEvent) {
		event.preventDefault();
		if (!canReserve) return;
		const submittingCustomerId = customer.id;
		try {
			const payload: HoldInput = pending?.payload ?? {
				variant_id: car.id,
				pickup_at: quote!.pickup_at,
				return_at: quote!.return_at,
				expected_total_idr: quote!.total_idr,
				terms_version: 'rental-terms-v1',
				accept_terms: true
			};
			pending = holdCommand(customer.id, car.slug, payload);
			const saved = await hold.mutateAsync(pending);
			if (
				!active ||
				client.getQueryData(customerProfileQuery.queryKey)?.id !== submittingCustomerId
			)
				return;
			toast({ kind: 'success', message: 'Reservation created. Continue to payment.' });
			await goto(resolve('/payment/[bookingId]', { bookingId: saved.id }));
		} catch (cause) {
			if (!active) return;
			const apiError = getApiError(cause);
			if (apiError?.code === 'QUOTE_CHANGED' || apiError?.code === 'AVAILABILITY_CONFLICT') {
				pending = undefined;
				clearPendingHold(customer.id, car.slug);
				accepted = false;
				quote = undefined;
				if (apiError.code === 'QUOTE_CHANGED') {
					const latest = isAxiosError<{ error: { details?: { quote?: RentalQuote } } }>(cause)
						? cause.response?.data.error.details?.quote
						: undefined;
					if (typeof latest === 'object' && latest !== null) quote = latest as RentalQuote;
				}
			}
			const messages: Record<string, string> = {
				QUOTE_CHANGED: 'The price has changed. Review the new total and accept the terms again.',
				AVAILABILITY_CONFLICT: 'This schedule is no longer available. Check another schedule.',
				BOOKING_IN_PROGRESS:
					'Your reservation is being processed. Retry with the same request in a moment.'
			};
			const message =
				messages[apiError?.code ?? ''] ??
				apiError?.message ??
				'Could not retrieve your reservation. Retry to check the same request.';
			feedback = { tone: 'error', message };
			toast({ kind: 'error', message });
		}
	}
</script>

<svelte:head>
	<title>Checkout — SROOM</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

{#snippet price()}
	{#if total === null}—{:else}{formatIDR(total)}{/if}
{/snippet}

{#snippet reserveButton(className: string)}
	<Button type="submit" disabled={!canReserve} aria-busy={reserving} class={className}>
		{reserving
			? 'Checking reservation…'
			: pending
				? 'Retry reservation'
				: 'Reserve & Continue to Payment'}
		<ArrowRight size={17} aria-hidden="true" class="shrink-0" />
	</Button>
{/snippet}

<div data-checkout class="page-container pt-7 sm:pt-10 lg:pt-14">
	<Link
		href={resolve('/cars')}
		variant="plain"
		class="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-ink-muted hover:text-ink"
	>
		<ArrowLeft size={17} aria-hidden="true" />
		Explore Cars
	</Link>

	<header class="mt-3 mb-8 sm:mb-10">
		<h1 class="text-4xl font-extrabold tracking-tight sm:text-5xl">Checkout</h1>
		<p class="mt-2 text-sm text-ink-muted sm:text-base">
			Choose your rental schedule for the {carName}.
		</p>
	</header>

	{#if pending}<p class="mb-5 max-w-2xl text-sm leading-6 text-ink-muted">
			An earlier reservation request is saved. Retry to retrieve its result, or change the schedule
			to start a new request.
		</p>{/if}
	<CustomerRequestState error={sessionError} label="session" />
	<form
		novalidate
		onsubmit={reserve}
		class="relative grid items-start gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:gap-12"
	>
		<div class="min-w-0">
			<section
				aria-labelledby="schedule-heading"
				class="rounded-card border border-line bg-panel p-4 sm:p-7"
			>
				<div class="flex items-start gap-3">
					<CalendarDays size={22} aria-hidden="true" class="mt-1 shrink-0 text-primary" />
					<div>
						<h2 id="schedule-heading" class="text-2xl font-bold tracking-tight">Rental schedule</h2>
						<p class="mt-1 text-sm leading-6 text-ink-muted">
							Pickup starts tomorrow. All times are in WITA (UTC+8).
						</p>
					</div>
				</div>

				<fieldset disabled={reserving} class="mt-7 min-w-0 space-y-5" oninput={scheduleChanged}>
					<div class="grid gap-4 sm:grid-cols-2">
						<TextField
							name="pickupDate"
							label="Pickup date"
							type="date"
							min={earliestPickup}
							bind:value={schedule.pickupDate}
							error={checked || touched.pickupDate ? errors.pickupDate : undefined}
							onblur={() => (touched.pickupDate = true)}
							required
						/>
						<TextField
							name="pickupTime"
							label="Pickup time"
							type="time"
							min="08:00"
							max="18:00"
							step="1800"
							bind:value={schedule.pickupTime}
							error={checked || touched.pickupTime ? errors.pickupTime : undefined}
							onblur={() => (touched.pickupTime = true)}
							required
						/>
					</div>
					<div class="grid gap-4 sm:grid-cols-2">
						<TextField
							name="returnDate"
							label="Return date"
							type="date"
							min={schedule.pickupDate || earliestPickup}
							bind:value={schedule.returnDate}
							error={checked || touched.returnDate ? errors.returnDate : undefined}
							onblur={() => (touched.returnDate = true)}
							required
						/>
						<TextField
							name="returnTime"
							label="Return time"
							type="time"
							min="08:00"
							max="18:00"
							step="1800"
							bind:value={schedule.returnTime}
							error={checked || touched.returnTime ? errors.returnTime : undefined}
							onblur={() => (touched.returnTime = true)}
							required
						/>
					</div>
				</fieldset>

				<div class="mt-6 border-t border-line pt-5">
					<Button
						variant="secondary"
						disabled={!ready || checking || reserving}
						onclick={(event: MouseEvent & { currentTarget: HTMLButtonElement }) =>
							void checkAvailability(event.currentTarget.form)}
					>
						{checking ? 'Checking availability…' : 'Check availability'}
					</Button>
					<div role="status" aria-atomic="true" tabindex="-1" class="focus:outline-none">
						{#if checking}
							<span class="sr-only">Checking availability for your schedule.</span>
						{:else if feedback}
							<p
								class={[
									'mt-3 flex items-start gap-2 text-sm leading-6',
									feedback.tone === 'success' && 'text-positive',
									feedback.tone === 'error' && 'text-negative'
								]}
							>
								{#if feedback.tone === 'success'}<CheckCircle2
										size={17}
										aria-hidden="true"
										class="mt-0.5 shrink-0"
									/>{/if}
								{feedback.message}
							</p>
						{/if}
					</div>
				</div>
			</section>

			<section
				aria-labelledby="pickup-heading"
				class="mt-7 flex gap-3 border-b border-line px-1 pb-6 sm:mt-8"
			>
				<MapPin size={20} aria-hidden="true" class="mt-0.5 shrink-0 text-primary" />
				<div>
					<h2 id="pickup-heading" class="text-lg font-bold">Pickup & return</h2>
					<p class="mt-1 text-sm text-ink-muted">Samarinda, East Kalimantan</p>
				</div>
			</section>

			<section aria-labelledby="renter-heading" class="mt-6 px-1">
				<h2 id="renter-heading" class="text-lg font-bold">Renter details</h2>
				<p class="mt-1 text-sm font-medium">{customer.full_name || customer.email}</p>
				<p class="text-sm break-all text-ink-muted">{customer.email}</p>
			</section>
		</div>

		<aside
			aria-labelledby="summary-heading"
			class="min-w-0 overflow-hidden rounded-card border border-line bg-panel lg:sticky lg:top-28"
		>
			{#if car.cover?.url}<img
					src={car.cover?.url ?? ''}
					alt={carName}
					class="h-40 w-full bg-panel-muted object-contain p-2 lg:h-44"
				/>{/if}
			<div class="p-4 sm:p-6">
				<h2 id="summary-heading" class="text-xl font-bold tracking-tight">{carName}</h2>
				<p class="mt-1 text-sm text-ink-muted">{car.trim} · {car.category}</p>

				<dl class="mt-5 space-y-3 border-t border-line pt-4 text-sm">
					<div class="flex items-baseline justify-between gap-3">
						<dt class="text-ink-muted">Daily rate</dt>
						<dd class="font-medium tabular-nums">
							{formatIDR(quote?.daily_rate_idr ?? car.daily_rate_idr)}
						</dd>
					</div>
					<div class="flex items-baseline justify-between gap-3">
						<dt class="text-ink-muted">Rental days</dt>
						<dd class="font-medium tabular-nums">{days ?? '—'}</dd>
					</div>
					<div class="flex items-baseline justify-between gap-3 border-t border-line pt-4">
						<dt class="font-semibold">Total</dt>
						<dd class="text-xl font-bold tabular-nums">{@render price()}</dd>
					</div>
				</dl>
				<p class="mt-2 text-sm leading-6 text-ink-muted">
					Pickup and return dates both count toward your rental.
				</p>

				<div class="mt-5 flex items-start gap-2 border-t border-line pt-4">
					<label class="flex min-h-11 items-start gap-3 py-2 text-sm leading-6">
						<input
							type="checkbox"
							name="terms"
							bind:checked={accepted}
							disabled={reserving}
							class="mt-1 size-4 shrink-0 accent-primary"
						/>
						<span>I agree to the Rental Terms.</span>
					</label>
					<Link
						href={resolve('/rental-terms')}
						target="_blank"
						rel="noopener"
						class="shrink-0 text-sm font-medium underline decoration-line hover:decoration-current"
						aria-label="Read the Rental Terms (opens in a new tab)">Read terms</Link
					>
				</div>
				<div class="mt-3 hidden lg:block">{@render reserveButton('w-full')}</div>
			</div>
		</aside>

		<div
			class="sticky bottom-0 z-10 -mx-4 border-t border-line bg-panel px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:-mx-6 sm:px-6 lg:hidden"
		>
			<div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
				<p class="flex flex-1 flex-wrap items-baseline gap-x-2 text-sm text-ink-muted">
					<span>Total</span>
					<strong class="text-base text-ink tabular-nums">{@render price()}</strong>
				</p>
				{@render reserveButton('min-w-0 grow')}
			</div>
		</div>
	</form>
</div>

<style>
	@media (width < 64rem) {
		:global(html:has([data-checkout])) {
			scroll-padding-bottom: calc(7rem + env(safe-area-inset-bottom));
		}
	}
</style>
