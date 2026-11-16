<script lang="ts">
	import { resolve } from '$app/paths';
	import CarCard from '#lib/components/cars/CarCard.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import type { CarPreview } from '#lib/types/car.ts';

	let { cars, unavailable }: { cars: CarPreview[]; unavailable: boolean } = $props();
</script>

<section id="fleet" class="bg-panel py-16 sm:py-24" aria-labelledby="fleet-heading">
	<div class="page-container">
		<div class="flex flex-col items-start justify-between gap-5 md:flex-row md:items-end md:gap-8">
			<div>
				<h2
					id="fleet-heading"
					class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight"
				>
					A car for your<br class="sm:hidden" /> kind of day.
				</h2>
				<p class="mt-4 max-w-lg text-base/7 text-pretty text-ink-muted">
					A little city run. A full family trip. A change of scenery.
				</p>
			</div>
			<Link href={resolve('/cars')} arrow class="shrink-0 font-medium">View all cars</Link>
		</div>
		{#if cars.length}
			<ul class="mt-8 grid gap-6 sm:mt-10 sm:grid-cols-2 lg:grid-cols-3" aria-label="Featured cars">
				{#each cars as car (car.id)}
					<li class="min-w-0"><CarCard {car} /></li>
				{/each}
			</ul>
		{:else}
			<p class="mt-8 rounded-card bg-panel-muted p-5 text-sm text-ink-muted">
				{unavailable
					? 'Featured cars are temporarily unavailable.'
					: 'There are no featured cars right now.'}
			</p>
		{/if}
	</div>
</section>
