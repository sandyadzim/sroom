<script lang="ts">
	import { ArrowUpRight } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import HeroArtwork from './HeroArtwork.svelte';

	const cars = [
		{
			make: 'Toyota',
			model: 'Fortuner',
			image: '/images/hero/fortuner-studio.webp',
			headline: ['Room for', 'bigger plans.'],
			description:
				'A spacious SUV for family days, longer drives, and a little more room along the way.'
		},
		{
			make: 'Honda',
			model: 'Brio',
			image: '/images/hero/brio-studio.webp',
			headline: ['Small car.', 'Big city energy.'],
			description:
				'A compact hatchback with a playful character, for everyday errands, coffee stops, and city drives.'
		},
		{
			make: 'Suzuki',
			model: 'Jimny',
			image: '/images/hero/jimny-studio.webp',
			headline: ['Boxy looks.', 'Weekend spirit.'],
			description:
				'A compact SUV with a distinctive boxy silhouette, for city detours and weekends with a little more character.'
		}
	];
	let active = $state(0);
	const selectedCar = $derived(cars[active]);
</script>

<section class="hero relative isolate overflow-hidden" aria-labelledby="hero-heading">
	{#each cars as car, index (car.model)}
		<div
			class={[
				'hero-background pointer-events-none absolute inset-0 transition-opacity duration-500 ease-in-out motion-reduce:transition-none',
				index === active ? 'opacity-100' : 'opacity-0'
			]}
		>
			<HeroArtwork {...car} priority={index === 0} />
		</div>
	{/each}

	<div class="pointer-events-none absolute inset-0 z-10">
		<div class="hero-copy page-container flex h-full items-end pb-28 sm:pb-32 lg:pb-10">
			<div class="pointer-events-auto max-w-80">
				<h1 id="hero-heading" class="text-3xl/9 font-semibold tracking-tight sm:text-4xl/10">
					<span class="sr-only">{selectedCar.make} {selectedCar.model}: </span>
					{selectedCar.headline[0]}<br />{selectedCar.headline[1]}
				</h1>
				<p class="mt-4 max-w-72 text-sm/6 text-ink-muted">
					{selectedCar.description}
				</p>
				<Button
					href="#"
					class="mt-6"
					aria-label={`Book Now — ${selectedCar.make} ${selectedCar.model}`}
				>
					Book Now <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
				</Button>
			</div>
		</div>
	</div>

	<div class="pointer-events-none absolute inset-x-0 bottom-4 z-20 sm:bottom-8 lg:bottom-10">
		<div class="page-container flex justify-center lg:justify-end">
			<div
				class="pointer-events-auto flex gap-2 sm:gap-3"
				role="group"
				aria-label="Choose a featured car"
			>
				{#each cars as car, index (car.model)}
					<Button
						variant="ghost"
						size="custom"
						aria-label={`Show ${car.make} ${car.model}`}
						aria-pressed={index === active}
						onclick={() => (active = index)}
						class={[
							'hero-thumbnail relative h-12 w-20 shrink-0 overflow-hidden rounded-lg border-2 sm:h-20 sm:w-28',
							index === active ? 'border-primary' : 'border-line hover:border-ink-muted'
						]}
					>
						<HeroArtwork {...car} thumbnail />
					</Button>
				{/each}
			</div>
		</div>
	</div>
</section>
