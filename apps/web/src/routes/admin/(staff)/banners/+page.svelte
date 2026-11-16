<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { getApiError } from '#lib/api/staff.ts';
	import BannerCard from '#lib/components/admin/BannerCard.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import { bannerSlotsQuery } from '#lib/queries/banner.ts';

	const banners = createQuery(() => bannerSlotsQuery);
</script>

<svelte:head>
	<title>Banners — SROOM</title>
</svelte:head>

<div class="mb-7">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Banners</h1>
	<p class="mt-2 text-sm/6 text-ink-muted">
		Manage the three positions in the homepage hero. Slot numbers set the display order.
	</p>
</div>

{#if banners.data?.length}
	<div class="@container">
		<ol aria-label="Homepage banner slots" class="grid gap-4 @2xl:grid-cols-3">
			{#each banners.data as slot (slot.position)}
				<li class="min-w-0"><BannerCard {slot} /></li>
			{/each}
		</ol>
	</div>
{:else if banners.isPending}
	<p role="status" class="rounded-xl border border-line bg-page px-4 py-5 text-sm text-ink-muted">
		Loading banner slots…
	</p>
{:else if banners.isError}
	<div class="rounded-xl border border-line bg-page px-4 py-5">
		<p role="alert" class="text-sm text-negative">
			{getApiError(banners.error)?.message ?? 'Could not load banner slots.'}
		</p>
		<Button
			class="mt-3"
			variant="secondary"
			disabled={banners.isFetching}
			onclick={() => banners.refetch()}
		>
			Try again
		</Button>
	</div>
{:else}
	<p class="rounded-xl border border-line bg-page px-4 py-5 text-sm text-ink-muted">
		No banner slots were returned.
	</p>
{/if}
