<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminShell from '#lib/components/admin/AdminShell.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import { staffProfileQuery } from '#lib/queries/identity.ts';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const profile = createQuery(() => staffProfileQuery);
	const role = $derived(profile.data?.role);
	const adminAccountsRoute = '/admin/(staff)/admins';
	const inAdminAccounts = $derived(
		page.route.id === adminAccountsRoute || page.route.id?.startsWith(`${adminAccountsRoute}/`)
	);

	$effect(() => {
		if (role === 'CUSTOMER') {
			void goto(resolve('/admin/login'), { replaceState: true });
		}
	});
</script>

{#if profile.isSuccess && (role === 'OWNER' || role === 'ADMIN')}
	<AdminShell profile={profile.data}>
		{#if inAdminAccounts && role !== 'OWNER'}
			<h1 class="text-3xl font-bold">Access denied</h1>
			<p class="mt-2 text-ink-muted">This page is for the owner.</p>
		{:else}
			{@render children()}
		{/if}
	</AdminShell>
{:else}
	<main id="main-content" tabindex="-1" class="page-container py-10">
		{#if profile.isError}
			<p role="alert" class="text-negative">
				{getApiError(profile.error)?.message ?? 'Could not load your profile.'}
			</p>
			<Button class="mt-3" variant="secondary" onclick={() => profile.refetch()}>Try again</Button>
		{:else}
			<p role="status" class="text-ink-muted">Loading staff workspace…</p>
		{/if}
	</main>
{/if}
