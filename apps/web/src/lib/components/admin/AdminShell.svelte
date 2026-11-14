<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { Menu, PanelLeftClose, PanelLeftOpen } from '@lucide/svelte';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { logout } from '#lib/api/identity/auth.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import type { Snippet } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import type { UserProfile } from '#lib/types/identity.ts';
	import Wordmark from '#lib/components/layout/Wordmark.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Dialog from '#lib/components/ui/Dialog.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import AdminNavigation from './AdminNavigation.svelte';

	let {
		profile,
		children
	}: { profile: Pick<UserProfile, 'role' | 'full_name' | 'email'>; children: Snippet } = $props();
	const profileHref = $derived(resolve('admin/profile'));
	const websiteHref = $derived(resolve('/'));
	const desktop = new MediaQuery('(min-width: 64rem)');
	const isOwner = $derived(profile.role === 'OWNER');
	const roleLabel = $derived(isOwner ? 'Owner' : 'Admin');
	const initials = $derived(
		(profile.full_name.trim() || profile.email)
			.split(/\s+/)
			.slice(0, 2)
			.map((word) => Array.from(word)[0]?.toUpperCase())
			.join('')
	);
	let menuOpen = $state(false);
	let collapsed = $state(false);

	const queryClient = useQueryClient();
	const showToast = useToast();
	const signOut = createMutation(() => ({
		mutationFn: logout,
		onSuccess: async () => {
			await goto(resolve('/admin/login'), { replaceState: true });
			queryClient.clear();
		},
		onError: (cause) => {
			showToast({ kind: 'error', message: getApiError(cause)?.message ?? 'Could not sign out.' });
		}
	}));

	$effect(() => {
		if (desktop.current) menuOpen = false;
	});
</script>

<div
	class={[
		'min-h-dvh bg-page lg:grid motion-safe:lg:transition-[grid-template-columns] motion-safe:lg:duration-200 motion-safe:lg:ease-[cubic-bezier(0.32,0.72,0,1)]',
		collapsed ? 'lg:grid-cols-[4.5rem_minmax(0,1fr)]' : 'lg:grid-cols-[15rem_minmax(0,1fr)]'
	]}
>
	<aside
		id="staff-sidebar"
		class="sticky top-0 hidden h-dvh flex-col overflow-x-hidden overflow-y-auto px-4 pt-4 pb-2 lg:flex"
	>
		<div class="mb-8">
			<Link
				href={profileHref}
				variant="plain"
				aria-label="SROOM staff home"
				class="grid min-h-11 grid-cols-[minmax(0,1fr)] items-center rounded-lg px-3 font-display leading-none font-bold tracking-tight whitespace-nowrap text-primary focus-visible:-outline-offset-2"
			>
				<span
					aria-hidden="true"
					class={[
						'col-start-1 row-start-1 text-3xl motion-safe:transition-opacity motion-safe:duration-150',
						collapsed && 'opacity-0'
					]}>SROOM.</span
				>
				<span
					aria-hidden="true"
					class={[
						'col-start-1 row-start-1 w-fit text-2xl motion-safe:transition-opacity motion-safe:duration-150',
						!collapsed && 'opacity-0'
					]}>S.</span
				>
			</Link>
			<p
				class={[
					'mt-1 px-3 text-sm whitespace-nowrap text-ink-muted motion-safe:transition-opacity motion-safe:duration-150',
					collapsed && 'opacity-0'
				]}
				aria-hidden={collapsed}
			>
				Staff workspace
			</p>
		</div>
		<AdminNavigation
			{isOwner}
			{profileHref}
			{websiteHref}
			{collapsed}
			onSignOut={() => signOut.mutate()}
			signingOut={signOut.isPending}
		/>
	</aside>

	<div class="flex min-h-dvh min-w-0 flex-col lg:pr-2 lg:pb-2">
		<header
			class="sticky top-0 z-30 flex min-h-16 items-center justify-between gap-4 bg-page px-4 sm:px-6 lg:min-h-14 lg:px-8"
		>
			<div class="flex min-w-0 items-center gap-2 lg:hidden">
				<Dialog
					title="Staff workspace"
					description="Navigate the SROOM staff workspace."
					closeLabel="Close navigation"
					placement="left"
					bind:open={menuOpen}
				>
					{#snippet trigger({ props })}
						<Button {...props} variant="ghost" size="icon" aria-label="Open navigation">
							<Menu size={22} strokeWidth={1.5} aria-hidden="true" />
						</Button>
					{/snippet}
					<AdminNavigation
						{isOwner}
						{profileHref}
						{websiteHref}
						onSignOut={() => signOut.mutate()}
						signingOut={signOut.isPending}
						onNavigate={() => (menuOpen = false)}
					/>
				</Dialog>
				<Wordmark href={profileHref} aria-label="SROOM staff home" />
			</div>

			<div class="hidden lg:block">
				<Button
					variant="ghost"
					size="icon"
					aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
					title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
					aria-controls="staff-sidebar"
					aria-expanded={!collapsed}
					onclick={() => (collapsed = !collapsed)}
				>
					{#if collapsed}
						<PanelLeftOpen size={20} strokeWidth={1.5} aria-hidden="true" />
					{:else}
						<PanelLeftClose size={20} strokeWidth={1.5} aria-hidden="true" />
					{/if}
				</Button>
			</div>

			<Button
				href={profileHref}
				variant="ghost"
				size="custom"
				aria-label={`Staff profile (${roleLabel})`}
				class="shrink-0 rounded-lg px-2 sm:px-3"
			>
				<span
					aria-hidden="true"
					class="grid size-8 shrink-0 place-items-center rounded-full bg-primary text-xs font-semibold text-on-primary"
				>
					{initials}
				</span>
				{roleLabel}
			</Button>
		</header>

		<main
			id="main-content"
			tabindex="-1"
			class="min-w-0 flex-1 border-t border-line bg-panel focus-visible:-outline-offset-4 lg:rounded-xl lg:border"
		>
			<div class="page-container py-8 sm:py-10">
				{@render children()}
			</div>
		</main>
	</div>
</div>
