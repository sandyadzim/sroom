<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { onDestroy, onMount } from 'svelte';
	import { ArrowLeft, Copy, RefreshCw, Clock3 } from '@lucide/svelte';
	import type { Booking } from '#lib/types/booking.ts';
	import type { CustomerPayment } from '#lib/types/payment.ts';
	import { createVirtualAccount } from '#lib/api/payment.ts';
	import { getApiError } from '#lib/api/error.ts';
	import {
		canShowVirtualAccount,
		paymentDeadline,
		rentalOutcome,
		virtualAccountKey
	} from '#lib/utils/rental.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import BookingSummary from './BookingSummary.svelte';
	import CustomerRequestState from './CustomerRequestState.svelte';
	let {
		customerId,
		booking,
		payment,
		reload,
		checkPayment,
		checking
	}: {
		customerId: string;
		booking: Booking;
		payment: CustomerPayment;
		reload: () => Promise<void>;
		checkPayment: () => Promise<void>;
		checking: boolean;
	} = $props();
	const toast = useToast();
	const create = createMutation(() => ({
		mutationFn: (key: string) => createVirtualAccount(booking.id, key),
		retry: false
	}));
	let active = true;
	onDestroy(() => {
		active = false;
	});
	let now = $state(Date.now());
	let started = $state(false);
	let copyMessage = $state('');
	let localError = $state('');
	const outcome = $derived(rentalOutcome(booking, payment));
	const deadline = $derived(paymentDeadline(payment));
	const remaining = $derived(Math.max(0, Math.ceil((deadline - now) / 1000)));
	const countdown = $derived(
		`${String(Math.floor(remaining / 60)).padStart(2, '0')}:${String(remaining % 60).padStart(2, '0')}`
	);
	const payable = $derived(
		booking.status === 'pending_payment' && !payment.review_required && now < deadline
	);
	const showInstructions = $derived(payable && canShowVirtualAccount(payment, now));
	const sessionError = $derived(
		isAxiosError(create.error) && create.error.response?.status === 401 ? create.error : undefined
	);
	$effect(() => {
		if (outcome)
			void goto(resolve('/bookings/[bookingId]/[outcome]', { bookingId: booking.id, outcome }), {
				replace: true
			});
	});
	onMount(() => {
		const timer = setInterval(() => {
			now = Date.now();
		}, 1000);
		return () => clearInterval(timer);
	});
	$effect(() => {
		if (!started && payment.status === 'not_started' && payable && !checking && !outcome) {
			started = true;
			void prepareVirtualAccount();
		}
	});
	async function prepareVirtualAccount() {
		if (create.isPending || !payable) return;
		localError = '';
		try {
			await create.mutateAsync(virtualAccountKey(customerId, booking.id));
			if (!active) return;
			await reload();
		} catch (cause) {
			if (!active) return;
			localError =
				getApiError(cause)?.message ??
				'Could not retrieve your Virtual Account. Check payment status or retry the same request.';
			toast({ kind: 'error', message: localError });
			await reload();
		}
	}
	async function retryVirtualAccount() {
		await reload();
		if (payment.status === 'not_started' && payable) await prepareVirtualAccount();
	}
	async function copyNumber() {
		if (!showInstructions || !payment.va_number) return;
		try {
			await navigator.clipboard.writeText(payment.va_number);
			copyMessage = 'Virtual Account number copied.';
		} catch {
			copyMessage = 'Could not copy. Select the number to copy it manually.';
		}
	}
</script>

<svelte:head
	><title>Payment — SROOM</title><meta name="robots" content="noindex, nofollow" /></svelte:head
>
<Link
	href={resolve('/account/bookings/[bookingId]', { bookingId: booking.id })}
	variant="plain"
	class="inline-flex min-h-11 items-center gap-2 text-sm text-ink-muted"
	><ArrowLeft size={17} aria-hidden="true" />View Booking</Link
>
<header class="mt-3 mb-8 sm:mb-10">
	<h1 class="text-4xl font-extrabold tracking-tight sm:text-5xl">Payment</h1>
	<p class="mt-2 text-ink-muted">Pay in full by BNI Virtual Account.</p>
</header>
<CustomerRequestState error={sessionError} label="session" />
<div class="grid items-start gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:gap-12">
	<section
		aria-labelledby="payment-heading"
		class="min-w-0 rounded-card border border-line bg-panel p-4 sm:p-7"
	>
		<h2 id="payment-heading" class="text-2xl font-bold tracking-tight">Bank transfer</h2>
		<div
			class="mt-5 flex flex-wrap items-center justify-between gap-4 rounded-xl bg-panel-muted px-4 py-3"
		>
			<div class="flex items-start gap-2">
				<Clock3 size={19} class="mt-0.5 shrink-0" aria-hidden="true" />
				<div>
					<p class="text-sm font-semibold">
						{remaining ? 'Payment deadline' : 'Payment deadline reached'}
					</p>
					<time datetime={new Date(deadline).toISOString()} class="text-sm text-ink-muted"
						>{formatWitaDateTime(new Date(deadline).toISOString())} WITA</time
					>
				</div>
			</div>
			<span
				role="timer"
				aria-label="Time remaining"
				class="font-display text-3xl font-bold tabular-nums">{countdown}</span
			>
		</div>
		<p class="mt-6 text-sm text-ink-muted">Total due</p>
		<p class="mt-1 text-3xl font-bold tabular-nums">{formatIDR(payment.amount_idr)}</p>
		<div class="mt-6 border-t border-line pt-5" role="status" aria-live="polite">
			{#if remaining === 0}<p class="text-sm leading-6 text-ink-muted">
					The payment deadline has passed. We’re waiting for the latest reservation status. Do not
					transfer to the previous Virtual Account.
				</p>
			{:else if payment.status === 'paid'}<p class="text-sm leading-6 text-ink-muted">
					Payment verified. Waiting for booking confirmation.
				</p>
			{:else if create.isPending}<p class="text-sm text-ink-muted">
					Preparing your Virtual Account…
				</p>
			{:else if showInstructions}
				<p class="text-sm font-semibold">BNI Virtual Account</p>
				<div class="mt-2 flex flex-wrap items-center gap-3">
					<p class="text-xl font-bold tracking-wide wrap-anywhere tabular-nums sm:text-2xl">
						{payment.va_number}
					</p>
					<Button
						variant="ghost"
						size="icon"
						aria-label="Copy Virtual Account number"
						onclick={() => void copyNumber()}><Copy size={17} aria-hidden="true" /></Button
					>
				</div>
				<p class="mt-2 text-sm text-ink-muted">
					Transfer the exact total before the deadline, then check payment status.
				</p>
				<p class="mt-2 text-sm text-ink-muted">{copyMessage}</p>
			{:else if payment.status === 'initiating' || payment.status === 'pending'}<p
					class="text-sm leading-6 text-ink-muted"
				>
					Your payment instructions are being checked. Check payment status to retrieve the latest
					details.
				</p>
			{:else if payment.status === 'failed'}<p class="text-sm leading-6 text-ink-muted">
					Payment could not be completed. Check the latest status before taking further action.
				</p>
			{:else}<p class="text-sm leading-6 text-ink-muted">
					Your Virtual Account has not been prepared yet.
				</p>{/if}
		</div>
		{#if localError}<p role="alert" class="mt-4 text-sm leading-6 text-negative">
				{localError}
			</p>{/if}
		<div class="mt-6 flex flex-wrap gap-3">
			<Button
				disabled={checking || create.isPending}
				aria-busy={checking}
				onclick={() => void checkPayment()}
				><RefreshCw size={17} aria-hidden="true" />Check payment status</Button
			>
			{#if payment.status === 'not_started' && payable && !create.isPending}<Button
					variant="secondary"
					disabled={checking}
					onclick={() => void retryVirtualAccount()}>Try again</Button
				>{/if}
		</div>
		{#if showInstructions}<section
				class="mt-7 border-t border-line pt-5"
				aria-labelledby="instructions-title"
			>
				<h3 id="instructions-title" class="font-semibold">How to pay</h3>
				<ol class="mt-3 list-decimal space-y-2 pl-5 text-sm leading-6 text-ink-muted">
					<li>Open your bank’s transfer or Virtual Account payment menu.</li>
					<li>Enter the BNI Virtual Account number shown above and verify the payment details.</li>
					<li>Complete the transfer for the exact total before the deadline.</li>
					<li>Return here and select Check payment status.</li>
				</ol>
			</section>{/if}
	</section>
	<BookingSummary {booking} paymentReference={payment.payment_id ?? undefined} />
</div>
