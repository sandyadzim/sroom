<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import type { CustomerSessionProfile } from '#lib/types/identity.ts';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { customerReturnPath } from '#lib/utils/customer-auth.ts';
	import Button from '#lib/components/ui/Button.svelte';
	let { children }: { children: Snippet<[CustomerSessionProfile]> } = $props();
	const profile = createQuery(() => ({ ...customerProfileQuery, refetchOnMount: 'always' }));
	$effect(() => {
		if (profile.data === null && !profile.isFetching) {
			const next = customerReturnPath(page.url.pathname);
			void goto(
				next ? resolve(`/(auth)/login?next=${encodeURIComponent(next)}`) : resolve('/(auth)/login'),
				{
					replace: true
				}
			);
		}
	});
</script>

{#if profile.isError}
	<section class="page-container py-10" aria-label="Customer session">
		<p role="alert" class="text-ink-muted">We couldn’t verify your session. Please try again.</p>
		<Button
			class="mt-4"
			variant="secondary"
			disabled={profile.isFetching}
			onclick={() => void profile.refetch()}>Try again</Button
		>
	</section>
{:else if !profile.data}
	<p role="status" class="page-container py-10 text-sm text-ink-muted">
		{profile.data === null && !profile.isFetching ? 'Opening sign in…' : 'Checking your session…'}
	</p>
{:else}
	{#key profile.data.id}{@render children(profile.data)}{/key}
{/if}
