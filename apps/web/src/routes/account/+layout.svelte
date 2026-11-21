<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { CalendarDays, LockKeyhole, UserRound } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { customerReturnPath } from '#lib/utils/customer-auth.ts';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const profile = createQuery(() => ({ ...customerProfileQuery, refetchOnMount: 'always' }));
	const profilePath = $derived(resolve('/account/profile'));
	const securityPath = $derived(resolve('/account/profile/change-password'));
	const links = $derived([
		{ label: 'Profile', href: profilePath, icon: UserRound },
		{ label: 'Security', href: securityPath, icon: LockKeyhole },
		{ label: 'My Bookings', href: resolve('/account/bookings'), icon: CalendarDays }
	]);

	$effect(() => {
		if (profile.data === null && !profile.isFetching) {
			const next =
				customerReturnPath(page.url.pathname + page.url.search) ??
				customerReturnPath(page.url.pathname) ??
				'/account/profile';
			void goto(resolve(`/(auth)/login?next=${encodeURIComponent(next)}`), { replace: true });
		}
	});
</script>

<svelte:head>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div data-account-shell class="page-container py-10 sm:py-14 lg:py-18">
	<div class="mx-auto max-w-5xl">
		{#if profile.data && !profile.isError}
			<div
				data-account-grid
				class="grid gap-8 md:grid-cols-[12rem_minmax(0,1fr)] md:gap-12 lg:grid-cols-[13rem_minmax(0,1fr)] lg:gap-16"
			>
				<nav
					aria-label="Account"
					class="border-b border-line pb-3 md:border-r md:border-b-0 md:pr-5 md:pb-0"
				>
					<ul class="flex flex-wrap gap-1 md:flex-col">
						{#each links as { label, href, icon: Icon } (href)}
							<li>
								<Link
									{href}
									variant="plain"
									aria-current={page.url.pathname === href ||
									(label === 'My Bookings' &&
										page.url.pathname.startsWith(resolve('/account/bookings')))
										? 'page'
										: undefined}
									class="flex min-h-11 items-center gap-2 rounded-lg px-2.5 text-sm font-medium text-ink-muted transition-colors hover:bg-panel-muted hover:text-ink aria-[current=page]:bg-panel-muted aria-[current=page]:text-primary motion-reduce:transition-none sm:px-3 md:gap-3"
								>
									<Icon size={17} aria-hidden="true" />
									{label}
								</Link>
							</li>
						{/each}
					</ul>
				</nav>
				<div class="min-w-0">
					{#key profile.data?.id}{@render children()}{/key}
				</div>
			</div>
		{:else if profile.isError && !profile.isFetching}
			<section aria-labelledby="account-title" class="space-y-4">
				<h1 id="account-title" class="text-3xl font-bold tracking-tight">Your account</h1>
				<p role="alert" class="text-ink-muted">We couldn’t load your profile. Please try again.</p>
				<Button variant="secondary" onclick={() => void profile.refetch()}>Try again</Button>
			</section>
		{:else}
			<p role="status" class="text-sm text-ink-muted">
				{profile.data === null && !profile.isFetching
					? 'Opening sign in…'
					: 'Loading your account…'}
			</p>
		{/if}
	</div>
</div>
