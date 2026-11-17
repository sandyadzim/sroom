<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { ChevronRight, MapPin, UsersRound, Gauge, Fuel, CarFront } from '@lucide/svelte';
	import CarGallery from '#lib/components/cars/CarGallery.svelte';
	import CarCard from '#lib/components/cars/CarCard.svelte';
	import RentalSummary from '#lib/components/cars/RentalSummary.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import type { load } from './+page.server.ts';

	let { data }: { data: Awaited<ReturnType<typeof load>> } = $props();
	const car = $derived(data.car);
	const name = $derived(`${car.brand} ${car.model}`);
	const profile = createQuery(() => customerProfileQuery);
	const bookingHref = $derived(
		car.status === 'active' && !profile.data
			? `${resolve('/(auth)/login')}?next=${encodeURIComponent(`/checkout/${car.slug}`)}`
			: undefined
	);
	const specs = $derived([
		{ label: 'Seats', value: `${car.seats} people`, icon: UsersRound },
		{ label: 'Transmission', value: car.transmission, icon: Gauge },
		{ label: 'Fuel', value: car.fuel, icon: Fuel },
		{ label: 'Body type', value: car.bodyType, icon: CarFront }
	]);
</script>

<svelte:head>
	<title>{name} {car.trim} — SROOM</title>
	<meta
		name="description"
		content={`Explore the ${name} ${car.trim}, a ${car.seats}-seat ${car.bodyType.toLowerCase()} for your next self-drive trip in Samarinda. View daily rates, specifications, and rental terms.`}
	/>
</svelte:head>

<div class="page-container pt-4 pb-8 sm:pt-8 lg:pt-10">
	<nav aria-label="Breadcrumb" class="mb-6 sm:mb-8">
		<ol class="flex flex-wrap items-center gap-x-2 text-sm/6 text-ink-muted">
			<li><Link href={resolve('/')} class="text-ink-muted">Home</Link></li>
			<li aria-hidden="true"><ChevronRight size={14} /></li>
			<li><Link href={resolve('/cars')} class="text-ink-muted">Explore cars</Link></li>
			<li aria-hidden="true"><ChevronRight size={14} /></li>
			<li><span aria-current="page">{name}</span></li>
		</ol>
	</nav>
	<header class="mb-6 sm:mb-8">
		<h1 class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight lg:text-[44px]/tight">
			{name}
		</h1>
		<p class="mt-2 text-base/7 text-ink-muted">
			{[car.trim, 'Self-drive in Samarinda'].filter(Boolean).join(' · ')}
		</p>
	</header>

	<div class="grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-10">
		<div class="min-w-0">
			{#key car.id}<CarGallery {name} images={car.gallery} />{/key}
			<section aria-labelledby="about-car" class="mt-10 sm:mt-12">
				<h2 id="about-car" class="text-2xl/8 font-semibold tracking-tight sm:text-3xl/tight">
					{name}
				</h2>
				{#if car.description}
					<p class="mt-4 max-w-2xl text-base/7 text-ink-muted">{car.description}</p>
				{/if}
				<dl class="mt-6 grid grid-cols-2 gap-x-6 gap-y-5 rounded-card bg-panel-muted p-5 sm:p-6">
					{#each specs as spec (spec.label)}
						<div class="min-w-0">
							<dt class="flex items-center gap-2 text-sm/6 text-ink-muted">
								<spec.icon size={18} strokeWidth={1.5} aria-hidden="true" />{spec.label}
							</dt>
							<dd class="mt-1 pl-6.5 text-base/7 font-medium">{spec.value}</dd>
						</div>
					{/each}
				</dl>
			</section>
			<section aria-labelledby="pickup-heading" class="mt-10 sm:mt-12">
				<h2 id="pickup-heading" class="text-2xl/8 font-semibold tracking-tight">
					Pickup in Samarinda.
				</h2>
				<p class="mt-4 flex items-center gap-2 text-base/7">
					<MapPin size={18} strokeWidth={1.5} aria-hidden="true" />Samarinda, East Kalimantan
				</p>
				<p class="mt-3 max-w-2xl text-base/7 text-ink-muted">
					Collect and return your car at the same pickup location. Rentals are self-drive, with no
					delivery or driver service. All pickup and return times use WITA (UTC+08:00).
				</p>
			</section>
		</div>
		<aside aria-label="Rental summary" class="lg:sticky lg:top-32">
			<RentalSummary
				{car}
				{bookingHref}
				bookingComingLater={car.status === 'active' && Boolean(profile.data)}
			/>
		</aside>
	</div>

	<section aria-labelledby="related-heading" class="mt-16 sm:mt-24">
		<div class="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-end">
			<h2
				id="related-heading"
				class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight"
			>
				A few more options.
			</h2>
			<Link href={resolve('/cars')} arrow class="shrink-0 font-medium">Explore all cars</Link>
		</div>
		{#if data.relatedCars.length > 0}
			<ul aria-label="Related cars" class="mt-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
				{#each data.relatedCars as related (related.id)}
					<li class="min-w-0"><CarCard car={related} /></li>
				{/each}
			</ul>
		{:else}
			<p class="mt-4 text-base/7 text-ink-muted">
				{data.relatedUnavailable
					? 'We couldn’t load other cars right now.'
					: 'There are no other cars to show right now.'}
			</p>
		{/if}
	</section>
</div>

<RentalSummary {car} {bookingHref} compact />
