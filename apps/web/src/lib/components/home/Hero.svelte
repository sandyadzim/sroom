<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowUpRight } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import type { PublicBanner } from '#lib/types/catalog.ts';
	import HeroArtwork from './HeroArtwork.svelte';

	let { banners }: { banners: PublicBanner[] } = $props();
	let active = $state(0);
	const selectedIndex = $derived(active < banners.length ? active : 0);
	const selectedBanner = $derived(banners[selectedIndex]);
</script>

{#if selectedBanner}
	<section class="hero relative isolate grid overflow-hidden" aria-labelledby="hero-heading">
		{#each banners as banner, index (index)}
			<div
				class={[
					'pointer-events-none absolute inset-0 transition-opacity duration-500 ease-in-out motion-reduce:transition-none',
					index === selectedIndex ? 'opacity-100' : 'opacity-0'
				]}
			>
				<HeroArtwork {...banner} priority={index === 0} />
			</div>
		{/each}

		<div
			class={[
				'relative z-10 page-container self-end pt-72 sm:pt-88 lg:pt-96',
				banners.length > 1 ? 'pb-28 sm:pb-32 lg:pb-10' : 'pb-10'
			]}
		>
			<div class="max-w-80">
				<h1
					id="hero-heading"
					class="text-3xl/9 font-semibold tracking-tight wrap-anywhere sm:text-4xl/10"
				>
					<span class="sr-only">{selectedBanner.make} {selectedBanner.model}: </span>
					{selectedBanner.headline[0]}<br />{selectedBanner.headline[1]}
				</h1>
				<p class="mt-4 max-w-72 text-sm/6 wrap-anywhere text-ink-muted">
					{selectedBanner.description}
				</p>
				<Button
					href={resolve('/cars/[slug]', { slug: selectedBanner.slug })}
					class="mt-6"
					aria-label={`Book Now — ${selectedBanner.make} ${selectedBanner.model}`}
				>
					Book Now <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
				</Button>
			</div>
		</div>

		{#if banners.length > 1}
			<div class="pointer-events-none absolute inset-x-0 bottom-4 z-20 sm:bottom-8 lg:bottom-10">
				<div class="page-container flex justify-center lg:justify-end">
					<div
						class="pointer-events-auto flex gap-2 sm:gap-3"
						role="group"
						aria-label="Choose a hero banner"
					>
						{#each banners as banner, index (index)}
							<Button
								variant="ghost"
								size="custom"
								aria-label={`Show ${banner.make} ${banner.model}: ${banner.headline.join(' ')}`}
								aria-pressed={index === selectedIndex}
								onclick={() => (active = index)}
								class={[
									'hero-thumbnail relative h-12 w-20 shrink-0 overflow-hidden rounded-lg border-2 sm:h-20 sm:w-28',
									index === selectedIndex ? 'border-primary' : 'border-line hover:border-ink-muted'
								]}
							>
								<HeroArtwork {...banner} thumbnail />
							</Button>
						{/each}
					</div>
				</div>
			</div>
		{/if}
	</section>
{:else}
	<section class="bg-panel py-20 sm:py-28" aria-labelledby="hero-heading">
		<div class="page-container">
			<h1 id="hero-heading" class="text-4xl/tight font-semibold tracking-tight sm:text-5xl/tight">
				Self-drive car rental in Samarinda
			</h1>
			<Button href={resolve('/cars')} class="mt-6">
				Explore Cars <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
			</Button>
		</div>
	</section>
{/if}
