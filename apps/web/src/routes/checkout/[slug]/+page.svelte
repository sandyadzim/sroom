<script lang="ts">
	import { asset, resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import { ArrowLeft, ArrowRight, CalendarDays, CheckCircle2, MapPin } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { inclusiveRentalDays, validateSchedule, type Schedule } from './schedule.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let ready = $state(false);
	onMount(() => {
		ready = true;
	});
	let schedule = $state<Schedule>({
		pickupDate: '',
		pickupTime: '',
		returnDate: '',
		returnTime: ''
	});
	let touched = $state<Partial<Record<keyof Schedule, boolean>>>({});
	let checked = $state(false);
	let accepted = $state(false);
	let checking = $state(false);
	let reserving = $state(false);
	let days = $state<number | null>(null);
	let feedback = $state<{ tone: 'success' | 'error' | 'info'; message: string }>();
	let availabilityStatus: HTMLDivElement;
	let version = 0;
	const errors = $derived(validateSchedule(schedule, data.earliestPickup));
	const valid = $derived(Object.values(errors).every((message) => !message));
	const total = $derived(days === null ? null : days * data.car.dailyRate);
	const canReserve = $derived(
		ready && valid && total !== null && accepted && !checking && !reserving
	);

	function scheduleChanged() {
		version += 1;
		checking = false;
		days = null;
		feedback = undefined;
	}

	async function checkAvailability(form: HTMLFormElement | null) {
		checked = true;
		const invalidField = (Object.keys(errors) as (keyof Schedule)[]).find((field) => errors[field]);
		if (invalidField) {
			form?.querySelector<HTMLInputElement>(`[name="${invalidField}"]`)?.focus();
			return;
		}
		if (checking || reserving) return;

		const current = ++version;
		days = null;
		feedback = undefined;
		checking = true;
		await new Promise((done) => setTimeout(done, 360));
		if (current !== version) return;
		checking = false;

		if (data.scenario === 'unavailable') {
			feedback = {
				tone: 'error',
				message: 'No cars available for this schedule. Try different dates or times.'
			};
		} else if (data.scenario === 'error') {
			feedback = { tone: 'error', message: 'We couldn’t check availability. Please try again.' };
		} else {
			days = inclusiveRentalDays(schedule.pickupDate, schedule.returnDate);
			feedback = { tone: 'success', message: 'Available for your selected schedule.' };
		}
	}

	async function reserve(event: SubmitEvent) {
		event.preventDefault();
		if (!canReserve) return;
		reserving = true;
		await new Promise((done) => setTimeout(done, 360));
		reserving = false;

		if (data.scenario === 'conflict') {
			days = null;
			feedback = {
				tone: 'error',
				message:
					'The car is no longer available for this schedule. Check availability again or choose another schedule.'
			};
		} else {
			feedback = { tone: 'info', message: 'Your schedule is ready for the payment step.' };
		}
		await tick();
		availabilityStatus?.focus();
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
	<Button type="submit" disabled={!canReserve} class={className}>
		{reserving ? 'Checking reservation…' : 'Reserve & Continue to Payment'}
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
			Choose your rental schedule for the {data.car.name}.
		</p>
	</header>

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
							min={data.earliestPickup}
							bind:value={schedule.pickupDate}
							error={checked || touched.pickupDate ? errors.pickupDate : undefined}
							onblur={() => (touched.pickupDate = true)}
							required
						/>
						<TextField
							name="pickupTime"
							label="Pickup time"
							type="time"
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
							min={schedule.pickupDate || data.earliestPickup}
							bind:value={schedule.returnDate}
							error={checked || touched.returnDate ? errors.returnDate : undefined}
							onblur={() => (touched.returnDate = true)}
							required
						/>
						<TextField
							name="returnTime"
							label="Return time"
							type="time"
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
					<div
						bind:this={availabilityStatus}
						role="status"
						aria-atomic="true"
						tabindex="-1"
						class="focus:outline-none"
					>
						{#if checking}
							<span class="sr-only">Checking availability for your schedule.</span>
						{:else if feedback}
							<p
								class={[
									'mt-3 flex items-start gap-2 text-sm leading-6',
									feedback.tone === 'success' && 'text-positive',
									feedback.tone === 'error' && 'text-negative',
									feedback.tone === 'info' && 'text-ink-muted'
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
				<p class="mt-1 text-sm font-medium">{data.customer.name}</p>
				<p class="text-sm break-all text-ink-muted">{data.customer.email}</p>
			</section>
		</div>

		<aside
			aria-labelledby="summary-heading"
			class="min-w-0 overflow-hidden rounded-card border border-line bg-panel lg:sticky lg:top-28"
		>
			<img
				src={asset(data.car.image)}
				alt={data.car.name}
				class="h-40 w-full bg-panel-muted object-contain p-2 lg:h-44"
			/>
			<div class="p-4 sm:p-6">
				<h2 id="summary-heading" class="text-xl font-bold tracking-tight">{data.car.name}</h2>
				<p class="mt-1 text-sm text-ink-muted">{data.car.trim} · {data.car.category}</p>

				<dl class="mt-5 space-y-3 border-t border-line pt-4 text-sm">
					<div class="flex items-baseline justify-between gap-3">
						<dt class="text-ink-muted">Daily rate</dt>
						<dd class="font-medium tabular-nums">{formatIDR(data.car.dailyRate)}</dd>
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
