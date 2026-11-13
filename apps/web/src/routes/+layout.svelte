<script lang="ts">
	import './layout.css';
	import favicon from '#lib/assets/favicon.svg';
	import SiteHeader from '#lib/components/layout/SiteHeader.svelte';
	import SiteFooter from '#lib/components/layout/SiteFooter.svelte';
	import AuthHeader from '#lib/components/auth/AuthHeader.svelte';
	import { page } from '$app/state';
	import Link from '#lib/components/ui/Link.svelte';
	import interFont from '@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?url';
	import manropeFont from '@fontsource-variable/manrope/files/manrope-latin-wght-normal.woff2?url';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const authPage = $derived(page.route.id?.startsWith('/(auth)/') ?? false);
	const adminPage = $derived(page.route.id?.startsWith('/admin/') ?? false);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="preload" href={interFont} as="font" type="font/woff2" crossorigin="anonymous" />
	<link rel="preload" href={manropeFont} as="font" type="font/woff2" crossorigin="anonymous" />
	{#if adminPage}<meta name="robots" content="noindex, nofollow" />{/if}
</svelte:head>

<div class="flex min-h-dvh flex-col">
	<Link
		variant="plain"
		href="#main-content"
		class="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-50 focus:rounded-lg focus:bg-primary focus:px-4 focus:py-3 focus:text-on-primary"
	>
		Skip to content
	</Link>

	{#if authPage}<AuthHeader />{:else if !adminPage}<SiteHeader />{/if}

	<main id="main-content" tabindex="-1" class="min-w-0 flex-1">
		{@render children()}
	</main>

	{#if !authPage && !adminPage}<SiteFooter />{/if}
</div>
