<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import BannerForm from '#lib/components/admin/BannerForm.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { bannerSlotQuery } from '#lib/queries/banner.ts';
	import type { PageProps } from './$types';

	let { params }: PageProps = $props();
	const position = $derived(/^[1-3]$/.test(params.position) ? Number(params.position) : 0);
	const banner = createQuery(() => bannerSlotQuery(position));
	const title = $derived(
		banner.data ? (banner.data.content ? 'Edit banner' : 'Add banner') : 'Banner slot'
	);
	const errorStatus = $derived(
		isAxiosError(banner.error) ? banner.error.response?.status : undefined
	);
</script>

<svelte:head>
	<title>{title} — SROOM</title>
</svelte:head>

<Link
	href={resolve('/admin/(staff)/banners')}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	Back to banners
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">
	{title}{#if position}<span class="text-ink-muted">{` · Slot ${position}`}</span>{/if}
</h1>

{#if !position || errorStatus === 404}
	<p class="mt-5 text-ink-muted" role="alert">Banner slot not found.</p>
{:else if banner.data}
	{#key banner.data.position}
		<BannerForm slot={banner.data} />
	{/key}
{:else if banner.isError}
	<p class="mt-5 text-negative" role="alert">
		{getApiError(banner.error)?.message ?? 'Could not load this banner slot.'}
	</p>
	{#if !errorStatus || errorStatus >= 500}
		<Button
			class="mt-3"
			variant="secondary"
			disabled={banner.isFetching}
			onclick={() => banner.refetch()}>Try again</Button
		>
	{/if}
{:else}
	<p class="mt-5 text-ink-muted" role="status">Loading banner slot…</p>
{/if}
