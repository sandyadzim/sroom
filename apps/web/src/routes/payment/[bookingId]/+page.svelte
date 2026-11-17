<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { ArrowLeft, Clock3, Copy, RefreshCw } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import BookingSummary from '#lib/components/booking/BookingSummary.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import {
		bookingOutcomeRoute,
		checkBookingPreview,
		checkoutPreviewSlug,
		readBookingPreview,
		remainingPaymentSeconds,
		type BookingOutcome
	} from '#lib/utils/booking-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const detailHref = $derived(
		resolve('/account/bookings/[bookingId]', { bookingId: data.bookingId })
	);
	const checkoutHref = $derived(resolve('/checkout/[slug]', { slug: checkoutPreviewSlug }));
	const vaNumber = '0000000000000000';
	const displayVaNumber = vaNumber.match(/.{4}/g)?.join(' ');

	let booking = $state<ReturnType<typeof readBookingPreview>>();
	let vaState = $state<'preparing' | 'ready' | 'error'>('preparing');
	let expiresAt = $state(0);
	let now = $state(0);
	let checkingStatus = $state(false);
	let openingOutcome = $state(false);
	let copyMessage = $state('');
	let statusMessage = $state<{ text: string; error?: boolean }>();
	let checkAttempts = 0;
	let prepareTimer: ReturnType<typeof setTimeout>;
	let statusTimer: ReturnType<typeof setTimeout>;
	const remaining = $derived(remainingPaymentSeconds(expiresAt, now));
	const countdown = $derived(
		`${String(Math.floor(remaining / 60)).padStart(2, '0')}:${String(remaining % 60).padStart(2, '0')}`
	);

	function prepareVirtualAccount(fail = false) {
		vaState = 'preparing';
		prepareTimer = setTimeout(() => {
			vaState = fail ? 'error' : 'ready';
		}, 420);
	}

	async function openOutcome(outcome: BookingOutcome) {
		openingOutcome = true;
		try {
			await goto(
				resolve('/bookings/[bookingId]/[outcome]', {
					bookingId: data.bookingId,
					outcome: bookingOutcomeRoute(outcome)
				}),
				{ replace: true }
			);
		} catch {
			openingOutcome = false;
			statusMessage = { text: 'Could not open the booking status. Please try again.', error: true };
		}
	}

	function checkStatus() {
		if (checkingStatus || openingOutcome) return;
		checkingStatus = true;
		statusMessage = undefined;
		checkAttempts += 1;
		statusTimer = setTimeout(() => {
			checkingStatus = false;
			if (data.scenario === 'status-error' && checkAttempts === 1) {
				statusMessage = { text: 'Could not check payment status. Please try again.', error: true };
				return;
			}
			try {
				const outcome = checkBookingPreview(data.bookingId);
				if (outcome) {
					void openOutcome(outcome);
					return;
				}
			} catch {
				statusMessage = { text: 'Could not check payment status. Please try again.', error: true };
				return;
			}
			statusMessage = {
				text:
					remaining > 0
						? 'No payment has been recorded yet.'
						: 'The reservation status is still being checked. Check again in a moment.'
			};
		}, 420);
	}

	async function copyNumber() {
		try {
			await navigator.clipboard.writeText(vaNumber);
			copyMessage = 'Virtual Account number copied.';
		} catch {
			copyMessage = 'Could not copy. Select the number to copy it manually.';
		}
	}

	$effect(() => {
		if (remaining === 0) copyMessage = '';
	});

	onMount(() => {
		booking = readBookingPreview(data.bookingId);
		if (!booking) return;
		expiresAt = booking.expiresAt;
		now = data.scenario === 'deadline' ? Math.max(Date.now(), expiresAt) : Date.now();
		if (booking.outcome) {
			void openOutcome(booking.outcome);
			return;
		}
		prepareVirtualAccount(data.scenario === 'va-error');

		let clock: ReturnType<typeof setInterval>;
		if (remainingPaymentSeconds(expiresAt, now) === 0) {
			checkStatus();
		} else {
			clock = setInterval(() => {
				now = Date.now();
				if (remainingPaymentSeconds(expiresAt, now) > 0) return;
				clearInterval(clock);
				checkStatus();
			}, 1000);
		}
		return () => {
			clearInterval(clock);
			clearTimeout(prepareTimer);
			clearTimeout(statusTimer);
		};
	});
</script>

<svelte:head>
	<title>Payment — SROOM</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="page-container py-8 sm:py-10 lg:py-14">
	<Link
		href={booking ? detailHref : checkoutHref}
		variant="plain"
		class="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-ink-muted hover:text-ink"
	>
		<ArrowLeft size={17} aria-hidden="true" />
		{booking ? 'View Booking' : 'Back to checkout'}
	</Link>

	<header class="mt-3 mb-8 sm:mb-10">
		<h1 class="text-4xl font-extrabold tracking-tight sm:text-5xl">Payment</h1>
		<p class="mt-2 text-sm text-ink-muted sm:text-base">Pay in full by Virtual Account.</p>
	</header>

	{#if openingOutcome}
		<p role="status" class="text-sm text-ink-muted">Opening booking status…</p>
	{:else if booking === undefined}
		<p role="status" class="text-sm text-ink-muted">Loading payment details…</p>
	{:else if booking === null}
		<section class="max-w-lg rounded-card border border-line bg-panel p-5 sm:p-7">
			<h2 class="text-xl font-bold">Payment details unavailable</h2>
			<p class="mt-2 text-sm leading-6 text-ink-muted">
				Open Checkout in this tab and choose a schedule to continue.
			</p>
			<Button href={checkoutHref} class="mt-5">Go to checkout</Button>
		</section>
	{:else}
		<div class="grid items-start gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:gap-12">
			<section
				aria-labelledby="payment-heading"
				class="min-w-0 overflow-hidden rounded-card border border-line bg-panel"
			>
				<div class="p-4 sm:p-7">
					<div class="flex flex-wrap items-center justify-between gap-3">
						<h2 id="payment-heading" class="text-2xl font-bold tracking-tight">Bank transfer</h2>
						<span class="rounded-full bg-caution/10 px-3 py-1 text-xs font-semibold text-caution">
							{remaining === 0 ? 'Awaiting status update' : 'Pending payment'}
						</span>
					</div>

					<div
						class="mt-5 flex flex-wrap items-center justify-between gap-x-5 gap-y-2 rounded-xl bg-panel-muted px-4 py-3 sm:px-5"
					>
						<div class="flex min-w-0 items-start gap-2.5">
							<Clock3 size={19} aria-hidden="true" class="mt-0.5 shrink-0 text-primary" />
							<div>
								<p class="text-sm font-semibold">
									{remaining === 0 ? 'Payment deadline reached' : 'Payment deadline'}
								</p>
								<time datetime={new Date(expiresAt).toISOString()} class="text-sm text-ink-muted">
									{formatWitaDateTime(new Date(expiresAt).toISOString())} WITA
								</time>
							</div>
						</div>
						<span
							role="timer"
							aria-label="Time remaining"
							class="font-display text-2xl font-bold text-primary tabular-nums sm:text-3xl"
							>{countdown}</span
						>
					</div>

					<p class="mt-6 text-sm text-ink-muted">Total due</p>
					<p
						class="mt-1 font-display text-3xl font-extrabold tracking-tight text-primary tabular-nums"
					>
						{formatIDR(booking.total)}
					</p>

					<div role="status" aria-atomic="true">
						{#if remaining === 0}
							<p class="mt-5 rounded-lg bg-caution/10 px-4 py-3 text-sm leading-6 text-caution">
								The payment deadline has passed. Wait for a status update before taking any further
								action.
							</p>
						{/if}
					</div>
					{#if remaining > 0 && vaState === 'preparing'}
						<p role="status" class="mt-5 border-t border-line pt-5 text-sm text-ink-muted">
							Preparing your Virtual Account…
						</p>
					{:else if remaining > 0 && vaState === 'error'}
						<div class="mt-5 border-t border-line pt-5">
							<p role="alert" class="text-sm text-negative">
								Could not load your Virtual Account. Try again to retrieve the payment instructions.
							</p>
							<Button variant="secondary" class="mt-3" onclick={() => prepareVirtualAccount()}
								>Try again</Button
							>
						</div>
					{:else if remaining > 0}
						<div class="mt-5 rounded-xl border border-line p-3 sm:p-4">
							<p class="text-sm font-semibold">Virtual Account number</p>
							<div class="mt-1 flex flex-wrap items-center justify-between gap-2">
								<p class="font-display text-lg font-bold tabular-nums select-all sm:text-xl">
									{displayVaNumber}
								</p>
								<Button
									variant="secondary"
									size="custom"
									class="shrink-0 rounded-full px-3.5 sm:px-5"
									aria-label="Copy number"
									onclick={copyNumber}
								>
									<Copy size={16} aria-hidden="true" />
									<span class="sm:hidden">Copy</span>
									<span class="hidden sm:inline">Copy number</span>
								</Button>
							</div>
							<p class="mt-1 text-xs leading-5 text-ink-muted">
								Example bank · Preview only. Do not transfer.
							</p>
							<p role="status" class={copyMessage ? 'mt-2 text-sm text-ink-muted' : undefined}>
								{copyMessage}
							</p>
						</div>

						<h3 class="mt-6 text-lg font-bold">How to pay</h3>
						<ol class="mt-2 list-decimal space-y-1.5 pl-5 text-sm leading-6 text-ink-muted">
							<li>Choose Virtual Account in your banking app.</li>
							<li>Enter the account number and review the total.</li>
							<li>
								Complete the transfer before the deadline, then return here to check its status.
							</li>
						</ol>
					{/if}
				</div>

				<div class="border-t border-line bg-panel-muted px-4 py-4 sm:px-7 sm:py-5">
					<Button
						variant="secondary"
						class="border border-line bg-panel aria-disabled:cursor-wait aria-disabled:opacity-60"
						aria-disabled={checkingStatus}
						onclick={checkStatus}
					>
						<RefreshCw size={16} aria-hidden="true" />
						{checkingStatus ? 'Checking payment status…' : 'Check Payment Status'}
					</Button>
					<div role="status" aria-atomic="true">
						{#if checkingStatus}
							<p class="mt-2 text-sm text-ink-muted">Checking the latest reservation status.</p>
						{:else if statusMessage}
							<p
								class={[
									'mt-2 text-sm leading-6',
									statusMessage.error ? 'text-negative' : 'text-ink-muted'
								]}
							>
								{statusMessage.text}
							</p>
						{/if}
					</div>
				</div>
			</section>

			<BookingSummary {booking} bookingId={data.bookingId} class="lg:sticky lg:top-28" />
		</div>
	{/if}
</div>
