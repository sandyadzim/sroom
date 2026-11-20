<script lang="ts">
	import { isAxiosError } from 'axios';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { customerRentalKey } from '#lib/queries/rental.ts';
	import Button from '#lib/components/ui/Button.svelte';
	let {
		error,
		loading = false,
		label = 'booking',
		retry
	}: { error?: unknown; loading?: boolean; label?: string; retry?: () => void } = $props();
	const client = useQueryClient();
	const status = $derived(isAxiosError(error) ? error.response?.status : undefined);
	$effect(() => {
		if (status === 401) {
			void client.cancelQueries({ queryKey: customerRentalKey });
			client.removeQueries({ queryKey: customerRentalKey });
			client.setQueryData(customerProfileQuery.queryKey, null);
		}
	});
</script>

{#if loading}<p role="status" class="py-6 text-sm text-ink-muted">Loading {label}…</p>
{:else if error}
	<section class="space-y-4 py-6">
		<h2 class="text-xl font-bold">
			{status === 401
				? 'Opening sign in…'
				: status === 404
					? `${label.charAt(0).toUpperCase() + label.slice(1)} unavailable`
					: status === 403
						? 'Access denied'
						: `Could not load your ${label}`}
		</h2>
		{#if status !== 401}<p role="alert" class="text-sm text-ink-muted">
				{status === 404
					? 'This record could not be found.'
					: status === 403
						? 'Use the customer account that owns this booking.'
						: 'Please try again in a moment.'}
			</p>{/if}
		{#if retry && status !== 401 && status !== 403 && status !== 404}<Button
				variant="secondary"
				onclick={retry}>Try again</Button
			>{/if}
	</section>
{/if}
