<script lang="ts">
	import CustomerSession from '#lib/components/auth/CustomerSession.svelte';
	import BookingRecord from '#lib/components/booking/BookingRecord.svelte';
	import Payment from '#lib/components/booking/Payment.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<div class="page-container py-8 sm:py-10 lg:py-14">
	<CustomerSession
		>{#snippet children(customer)}
			<BookingRecord customerId={customer.id} bookingId={data.bookingId}>
				{#snippet children({ booking, payment, reload, checkPayment, checking })}
					<Payment
						{booking}
						{payment}
						{checkPayment}
						{checking}
						customerId={customer.id}
						{reload}
					/>
				{/snippet}
			</BookingRecord>
		{/snippet}</CustomerSession
	>
</div>
