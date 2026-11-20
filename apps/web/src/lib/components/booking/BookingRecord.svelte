<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { onDestroy, type Snippet } from 'svelte';
	import { bookingQuery, paymentQuery } from '#lib/queries/rental.ts';
	import { refreshCustomerPayment } from '#lib/api/payment.ts';
	import { getApiError } from '#lib/api/error.ts';
	import type { Booking } from '#lib/types/booking.ts';
	import type { CustomerPayment } from '#lib/types/payment.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import CustomerRequestState from './CustomerRequestState.svelte';
	let {
		customerId,
		bookingId,
		children
	}: {
		customerId: string;
		bookingId: string;
		children: Snippet<
			[
				{
					booking: Booking;
					payment: CustomerPayment;
					reload: () => Promise<void>;
					checkPayment: () => Promise<void>;
					checking: boolean;
				}
			]
		>;
	} = $props();
	const client = useQueryClient();
	const toast = useToast();
	const booking = createQuery(() => bookingQuery(customerId, bookingId));
	const payment = createQuery(() => paymentQuery(customerId, bookingId));
	const refresh = createMutation(() => ({
		mutationFn: () => refreshCustomerPayment(bookingId),
		retry: false
	}));
	let active = true;
	onDestroy(() => {
		active = false;
	});
	let confirmationCycle = $state(0);
	let confirmationTimedOut = $state(false);
	async function reload() {
		if (!active) return;
		await Promise.all([booking.refetch(), payment.refetch()]);
	}
	async function checkPayment() {
		if (refresh.isPending || booking.isFetching || payment.isFetching) return;
		try {
			const result = await refresh.mutateAsync();
			if (!active) return;
			client.setQueryData(paymentQuery(customerId, bookingId).queryKey, result);
			await reload();
			confirmationCycle += 1;
			toast({
				kind: 'success',
				message:
					result.status === 'paid'
						? 'Payment verified. Checking booking confirmation.'
						: 'Payment status updated.'
			});
		} catch (cause) {
			toast({
				kind: 'error',
				message: getApiError(cause)?.message ?? 'Could not check payment. Please try again.'
			});
		}
	}
	const awaitingConfirmation = $derived(
		payment.data?.status === 'paid' &&
			booking.data?.status === 'pending_payment' &&
			!payment.data?.review_required &&
			!booking.isError &&
			!payment.isError
	);
	$effect(() => {
		void confirmationCycle;
		confirmationTimedOut = false;
		if (!awaitingConfirmation) return;
		let fetching = false;
		const timer = setInterval(async () => {
			if (fetching) return;
			fetching = true;
			try {
				await reload();
			} finally {
				fetching = false;
			}
		}, 3000);
		const stop = setTimeout(() => {
			clearInterval(timer);
			confirmationTimedOut = true;
		}, 30_000);
		return () => {
			clearInterval(timer);
			clearTimeout(stop);
		};
	});
	const sessionError = $derived(
		[booking.error, payment.error, refresh.error].find(
			(cause) => isAxiosError(cause) && cause.response?.status === 401
		)
	);
</script>

<CustomerRequestState error={sessionError} label="session" />
{#if booking.isError || payment.isError || !booking.data || !payment.data}
	<CustomerRequestState
		error={booking.error ?? payment.error}
		loading={booking.isPending || payment.isPending}
		retry={() => void reload()}
		label="booking details"
	/>
{:else}
	{#if payment.data.status === 'paid' && booking.data.status === 'pending_payment' && !payment.data.review_required}
		<p role="status" class="mb-6 rounded-xl bg-panel-muted p-4 text-sm leading-6 text-ink-muted">
			{confirmationTimedOut
				? 'Confirmation is taking longer than expected. Check payment status again in a moment.'
				: 'Processing confirmation. Your payment has been verified; we’re waiting for the booking confirmation.'}
		</p>
	{/if}
	{@render children({
		booking: booking.data,
		payment: payment.data,
		reload,
		checkPayment,
		checking: refresh.isPending || booking.isFetching || payment.isFetching
	})}
{/if}
