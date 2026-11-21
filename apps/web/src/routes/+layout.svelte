<script lang="ts">
	import './layout.css';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import SiteHeader from '#lib/components/layout/SiteHeader.svelte';
	import SiteFooter from '#lib/components/layout/SiteFooter.svelte';
	import AuthHeader from '#lib/components/auth/AuthHeader.svelte';
	import { navigating, page } from '$app/state';
	import Link from '#lib/components/ui/Link.svelte';
	import CustomerCache from '#lib/components/auth/CustomerCache.svelte';
	import Toaster from '#lib/components/ui/Toaster.svelte';
	import { setToastContext, type ToastFeedback, type ToastItem } from '#lib/ui/toast.ts';
	import interFont from '@fontsource-variable/inter/files/inter-latin-wght-normal.woff2?url';
	import manropeFont from '@fontsource-variable/manrope/files/manrope-latin-wght-normal.woff2?url';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const authPage = $derived(page.route.id?.startsWith('/(auth)/') ?? false);
	const adminPage = $derived(page.route.id === '/admin' || page.route.id?.startsWith('/admin/'));
	const staffPage = $derived(page.route.id?.startsWith('/admin/(staff)'));
	const transactionPage = $derived(
		['/checkout/', '/payment/', '/bookings/'].some((prefix) => page.route.id?.startsWith(prefix))
	);

	const customerQueryClient = new QueryClient();

	let nextToastId = 0;
	let toasts = $state.raw<ToastItem[]>([]);

	function showToast(feedback: ToastFeedback) {
		toasts = [{ ...feedback, id: nextToastId++ }, ...toasts].slice(0, 3);
	}

	function dismissToast(id: number) {
		toasts = toasts.filter((toast) => toast.id !== id);
	}

	setToastContext(showToast);
</script>

<svelte:head>
	<link rel="preload" href={interFont} as="font" type="font/woff2" crossorigin="anonymous" />
	<link rel="preload" href={manropeFont} as="font" type="font/woff2" crossorigin="anonymous" />
	{#if adminPage || transactionPage}<meta name="robots" content="noindex, nofollow" />{/if}
</svelte:head>

<div class="flex min-h-dvh flex-col">
	<Link
		variant="plain"
		href="#main-content"
		class="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-50 focus:rounded-lg focus:bg-primary focus:px-4 focus:py-3 focus:text-on-primary"
	>
		Skip to content
	</Link>

	{#if staffPage}
		{@render children()}
	{:else if adminPage}
		<main id="main-content" tabindex="-1" class="min-w-0 flex-1">{@render children()}</main>
	{:else}
		<QueryClientProvider client={customerQueryClient}>
			<CustomerCache />
			{#if authPage}<AuthHeader />{:else}<SiteHeader />{/if}

			{#if navigating.to?.route.id === '/cars/[slug]'}
				<p
					role="status"
					class="fixed right-4 bottom-4 z-40 rounded-full bg-panel-muted px-4 py-2 text-sm text-ink"
				>
					Loading car details…
				</p>
			{/if}
			<main
				id="main-content"
				tabindex="-1"
				class="min-w-0 flex-1"
				aria-busy={navigating.to?.route.id === '/cars/[slug]' || undefined}
			>
				{@render children()}
			</main>

			{#if !authPage}<SiteFooter />{/if}
		</QueryClientProvider>
	{/if}
</div>

<Toaster {toasts} onDismiss={dismissToast} />
