<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import BookingRecord from '#lib/components/booking/BookingRecord.svelte';
	import BookingDetail from '#lib/components/booking/BookingDetail.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	const profile = createQuery(() => customerProfileQuery);
</script>

{#if profile.data}
	<BookingRecord customerId={profile.data.id} bookingId={data.bookingId}>
		{#snippet children({ booking, payment, checkPayment, checking })}
			<BookingDetail {booking} {payment} {checkPayment} {checking} />
		{/snippet}
	</BookingRecord>
{/if}
