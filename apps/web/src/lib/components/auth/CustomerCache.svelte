<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { customerRentalKey } from '#lib/queries/rental.ts';
	const profile = createQuery(() => customerProfileQuery);
	const client = useQueryClient();
	$effect(() => {
		const customerId = profile.data?.id;
		if (profile.data === undefined) return;
		const filters = {
			predicate: (query: { queryKey: readonly unknown[] }) =>
				query.queryKey[0] === customerRentalKey[0] && query.queryKey[1] !== customerId
		};
		void client.cancelQueries(filters);
		client.removeQueries(filters);
	});
</script>
