<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import Link from '#lib/components/ui/Link.svelte';
	import { customerLookupQuery } from '#lib/queries/staff-transactions.ts';
	import { customerDisplayName } from '#lib/utils/staff-transactions.ts';

	let { customerId }: { customerId: string } = $props();
	const lookup = createQuery(() => customerLookupQuery([customerId]));
	const label = $derived(lookup.data?.[0] ? customerDisplayName(lookup.data[0]) : customerId);
</script>

<Link
	href={resolve('/admin/(staff)/customers/[id]', { id: customerId })}
	variant="plain"
	class="text-primary underline"
>
	{label}
</Link>
