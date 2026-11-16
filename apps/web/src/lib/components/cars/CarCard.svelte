<script lang="ts">
	import { resolve } from '$app/paths';
	import { UsersRound, Gauge } from '@lucide/svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import type { CarPreview } from '#lib/types/car.ts';
	import { formatIDR } from '#lib/utils/currency.ts';

	let { car }: { car: CarPreview } = $props();
	let failedImage = $state<string | null>(null);
</script>

<article class="min-w-0">
	<Link
		href={car.href ?? resolve('/cars/[slug]', { slug: car.id })}
		variant="plain"
		aria-label={`View details for ${car.brand} ${car.model} ${car.trim}`}
		class="isolate block overflow-hidden rounded-card bg-panel-muted focus-visible:-outline-offset-4"
	>
		<div class="relative aspect-4/3 overflow-hidden">
			{#if !car.image || failedImage === car.image}
				<div
					role="img"
					aria-label={`Image unavailable for ${car.brand} ${car.model}`}
					class="flex h-full flex-col items-center justify-center gap-2 px-5 text-center"
				>
					<span class="font-display text-xl font-semibold">{car.brand} {car.model}</span>
					<span class="text-sm text-ink-muted">Image unavailable</span>
				</div>
			{:else}
				<img
					src={car.image}
					alt={car.imageAlt || `${car.brand} ${car.model}`}
					width="800"
					height="600"
					loading="lazy"
					onerror={() => (failedImage = car.image)}
					class="h-full w-full object-cover transition-transform duration-200 ease-out motion-safe:group-hover:scale-105 motion-safe:group-focus-visible:scale-105 motion-reduce:transition-none"
				/>
			{/if}
		</div>
		<div class="px-5 pt-4 pb-6">
			<h3 class="text-xl/7 font-semibold tracking-tight">{car.brand} {car.model}</h3>
			<p class="mt-1 text-sm/6 text-ink-muted">{car.trim}</p>
			<ul class="mt-4 flex flex-wrap gap-x-5 gap-y-2 text-sm/6 text-ink-muted">
				<li class="flex items-center gap-2">
					<UsersRound size={16} strokeWidth={1.5} aria-hidden="true" />{car.seats} seats
				</li>
				<li class="flex items-center gap-2">
					<Gauge size={16} strokeWidth={1.5} aria-hidden="true" />{car.transmission}
				</li>
			</ul>
			<div class="mt-6 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
				<p class="text-lg/7 font-semibold tabular-nums">
					{formatIDR(car.dailyRate)}<span class="ml-1 text-sm font-normal text-ink-muted">
						/ day
					</span>
				</p>
				<p class="text-sm/6 text-ink-muted">Self-drive</p>
			</div>
		</div>
	</Link>
</article>
