<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import InvoicePage from '#lib/components/booking/InvoicePage.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	const profile = createQuery(() => customerProfileQuery);
</script>

{#if profile.data}
	<InvoicePage customerId={profile.data.id} bookingId={data.bookingId} />
{/if}
