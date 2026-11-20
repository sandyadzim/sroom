<script lang="ts">
	import CustomerSession from '#lib/components/auth/CustomerSession.svelte';
	import Checkout from '#lib/components/booking/Checkout.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<CustomerSession
	>{#snippet children(customer)}{#key data.car.id}<Checkout
				car={data.car}
				{customer}
			/>{/key}{/snippet}</CustomerSession
>
