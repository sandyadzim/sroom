<script lang="ts">
	import CustomerSession from '#lib/components/auth/CustomerSession.svelte';
	import BookingRecord from '#lib/components/booking/BookingRecord.svelte';
	import BookingOutcome from '#lib/components/booking/BookingOutcome.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<div class="page-container py-8 sm:py-10 lg:py-14">
	<CustomerSession
		>{#snippet children(customer)}
			<BookingRecord customerId={customer.id} bookingId={data.bookingId}>
				{#snippet children({ booking, payment, checkPayment, checking })}
					<BookingOutcome
						{booking}
						{payment}
						{checkPayment}
						{checking}
						requestedOutcome={data.outcome}
					/>
				{/snippet}
			</BookingRecord>
		{/snippet}</CustomerSession
	>
</div>
