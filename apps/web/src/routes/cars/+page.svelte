<script lang="ts">
	import { tick } from 'svelte';
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { navigating } from '$app/state';
	import CarCard from '#lib/components/cars/CarCard.svelte';
	import CatalogFilterBar from '#lib/components/cars/CatalogFilters.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Pagination from '#lib/components/ui/Pagination.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import type { CatalogFilters } from '#lib/types/catalog.ts';
	import {
		catalogSorts,
		catalogUrlParams,
		defaultCatalogFilters,
		emptyCatalogFilters,
		readCatalogFilters
	} from '#lib/utils/catalog-filters.ts';
	import type { load } from './+page.server.ts';

	let { data }: { data: Awaited<ReturnType<typeof load>> } = $props();
	let resultsHeading: HTMLHeadingElement;
	const pagination = $derived(data.pagination);
	let retrying = $state(false);
	const loading = $derived(retrying || navigating.to?.route.id === '/cars');
	const hasFilters = $derived(
		Object.entries(data.filters).some(([key, value]) => key !== 'sort' && value)
	);

	function showPage(filters: CatalogFilters, page: number) {
		const params = catalogUrlParams(filters, page);
		return goto(resolve(params.size ? `/cars?${params}` : '/cars'), { reset: false });
	}

	function applyFilters(changes: Partial<CatalogFilters>) {
		const pendingUrl = loading ? navigating.to?.url : undefined;
		const filters = pendingUrl ? readCatalogFilters(pendingUrl.searchParams) : data.filters;
		return showPage({ ...filters, ...changes }, 1);
	}

	async function retry() {
		retrying = true;
		try {
			await refreshAll();
		} finally {
			retrying = false;
		}
	}

	async function changePage(value: number) {
		if (loading || value === pagination.page) return;
		await showPage(data.filters, value);
		await tick();
		resultsHeading.focus({ preventScroll: true });
		resultsHeading.scrollIntoView({ block: 'start' });
	}
</script>

<svelte:head>
	<title>Explore Cars — SROOM</title>
	<meta
		name="description"
		content="Explore self-drive cars in Samarinda with SROOM. Compare models, seats, and daily rates, then choose the car for your next drive."
	/>
</svelte:head>

<div class="page-container pt-8 pb-4 sm:pt-12 lg:pt-16">
	<header class="mb-6 sm:mb-8">
		<h1 class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight">Explore cars.</h1>
	</header>

	<CatalogFilterBar
		brands={data.brands}
		brandsUnavailable={data.brandsUnavailable}
		filters={data.filters}
		onapply={applyFilters}
	/>

	<section aria-labelledby="cars-results" class="mt-6 sm:mt-8">
		<div class="mb-6 flex flex-col justify-between gap-4 sm:flex-row sm:items-end sm:gap-6">
			<div>
				<h2
					id="cars-results"
					bind:this={resultsHeading}
					tabindex="-1"
					class="w-fit text-base/7 font-medium"
					aria-live="polite"
					aria-atomic="true"
				>
					{#if loading}
						Loading cars…
					{:else if data.unavailable}
						Could not load cars.
					{:else if data.total === 0}
						No cars found.
					{:else}
						Showing {pagination.start}{pagination.end === pagination.start
							? ''
							: `–${pagination.end}`} of {data.total}
						{data.total === 1 ? 'car' : 'cars'}
					{/if}
				</h2>
			</div>
			<div class="w-full sm:w-64">
				<SelectField
					name="sort"
					label="Sort by"
					options={catalogSorts}
					value={data.filters.sort}
					clearable={data.filters.sort !== defaultCatalogFilters.sort}
					onchange={(sort) => applyFilters({ sort: sort || defaultCatalogFilters.sort })}
				/>
			</div>
		</div>
		<div aria-busy={loading}>
			{#if data.unavailable}
				<div class="rounded-card bg-panel-muted p-6">
					<p class="text-sm/6 text-ink-muted">Please try loading the car list again.</p>
					<Button variant="secondary" class="mt-4" disabled={loading} onclick={retry}
						>Try again</Button
					>
				</div>
			{:else if data.cars.length === 0}
				<div class="rounded-card bg-panel-muted p-6">
					<p class="text-sm/6 text-ink-muted">
						{hasFilters ? 'No cars match your filters.' : 'There are no cars to browse right now.'}
					</p>
					{#if hasFilters}
						<Button
							variant="secondary"
							class="mt-4"
							onclick={() => applyFilters(emptyCatalogFilters)}>Clear filters</Button
						>
					{/if}
				</div>
			{:else}
				<ul class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3" aria-label="Rental cars">
					{#each data.cars as car (car.id)}
						<li class="min-w-0"><CarCard {car} /></li>
					{/each}
				</ul>
				{#if pagination.totalPages > 1}
					<div class="mt-8 flex flex-col items-center justify-between gap-4 sm:mt-10 sm:flex-row">
						<p class="text-sm/6 text-ink-muted">
							Page {pagination.page} of {pagination.totalPages}
						</p>
						<Pagination
							page={pagination.page}
							totalPages={pagination.totalPages}
							onchange={changePage}
						/>
					</div>
				{/if}
			{/if}
		</div>
	</section>
</div>
