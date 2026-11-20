<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { CircleCheck, Clock3, ShieldAlert, RefreshCw } from '@lucide/svelte';
	import type { Booking } from '#lib/types/booking.ts';
	import type { CustomerPayment } from '#lib/types/payment.ts';
	import { rentalOutcome } from '#lib/utils/rental.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import BookingSummary from './BookingSummary.svelte';
	let {
		booking,
		payment,
		requestedOutcome,
		checkPayment,
		checking
	}: {
		booking: Booking;
		payment: CustomerPayment;
		requestedOutcome: string;
		checkPayment: () => Promise<void>;
		checking: boolean;
	} = $props();
	const outcome = $derived(rentalOutcome(booking, payment));
	const content = {
		success: {
			title: 'Booking confirmed',
			description: 'Your payment is complete and your car is reserved for the dates shown.',
			icon: CircleCheck,
			tone: 'text-positive bg-positive/10'
		},
		expired: {
			title: 'Reservation expired',
			description:
				'This reservation has ended. Explore the available cars to choose a new schedule.',
			icon: Clock3,
			tone: 'text-negative bg-negative/10'
		},
		review: {
			title: 'Payment under review',
			description:
				'We’re checking your payment and reservation. Wait for an update before making any further payment.',
			icon: ShieldAlert,
			tone: 'text-information bg-information/10'
		}
	};
	$effect(() => {
		if (outcome === requestedOutcome) return;
		void goto(
			outcome
				? resolve('/bookings/[bookingId]/[outcome]', { bookingId: booking.id, outcome })
				: resolve('/payment/[bookingId]', { bookingId: booking.id }),
			{ replace: true }
		);
	});
</script>

<svelte:head
	><title>{outcome ? content[outcome].title : 'Booking status'} — SROOM</title><meta
		name="robots"
		content="noindex, nofollow"
	/></svelte:head
>
{#if outcome && outcome === requestedOutcome}
	{@const view = content[outcome]}
	<div
		class="mx-auto grid max-w-6xl items-start gap-8 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:gap-16"
	>
		<section class="min-w-0 lg:pt-5">
			<div class={['grid size-14 place-items-center rounded-2xl', view.tone]}>
				<view.icon size={28} aria-hidden="true" />
			</div>
			<h1 class="mt-5 text-4xl font-extrabold tracking-tight sm:text-5xl">{view.title}</h1>
			<p class="mt-3 max-w-lg leading-7 text-ink-muted">{view.description}</p>
			<div class="mt-7 flex flex-wrap gap-3 border-t border-line pt-6">
				{#if outcome === 'expired'}<Button href={resolve('/cars')}>Explore Cars</Button>
				{:else}
					<Button href={resolve('/account/bookings/[bookingId]', { bookingId: booking.id })}
						>View Booking</Button
					>
					{#if outcome === 'success'}<Button
							variant="secondary"
							href={resolve('/account/bookings/[bookingId]/invoice', { bookingId: booking.id })}
							>View Invoice</Button
						>
					{:else}<Button
							variant="secondary"
							disabled={checking}
							aria-busy={checking}
							onclick={() => void checkPayment()}
							><RefreshCw size={17} aria-hidden="true" />Check payment status</Button
						>{/if}
				{/if}
			</div>
		</section>
		<BookingSummary
			{booking}
			totalLabel={outcome === 'success' ? 'Total paid' : 'Booking total'}
			paymentReference={payment.payment_id ?? undefined}
		/>
	</div>
{:else}<p role="status" class="text-sm text-ink-muted">Opening the latest booking status…</p>{/if}
