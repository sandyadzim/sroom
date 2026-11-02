<script lang="ts">
	import { tick } from 'svelte';
	import CarCard from '#lib/components/cars/CarCard.svelte';
	import CatalogFilterBar from '#lib/components/cars/CatalogFilters.svelte';
	import Pagination from '#lib/components/ui/Pagination.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import { demoCatalogCars } from '#lib/data/demo-fleet.ts';
	import {
		defaultCatalogFilters,
		prepareCatalogRequest,
		type CatalogFilters
	} from '#lib/types/catalog.ts';
	import { getPagination } from '#lib/utils/pagination.ts';

	const sorts = [
		{ value: 'recommended', label: 'Recommended' },
		{ value: 'price-asc', label: 'Price: low to high' },
		{ value: 'price-desc', label: 'Price: high to low' },
		{ value: 'name-asc', label: 'Name: A to Z' }
	];
	const pageSize = 6;
	const brands = [...new Set(demoCatalogCars.map((car) => car.brand))].sort();
	let filters = $state({ ...defaultCatalogFilters });
	let currentPage = $state(1);
	let resultsHeading: HTMLHeadingElement;
	const pagination = $derived(getPagination(demoCatalogCars.length, currentPage, pageSize));
	const cars = $derived(demoCatalogCars.slice(pagination.offset, pagination.end));

	function requestCars() {
		// Replace this handler with the catalogue API request when its contract is approved.
		// Filtering and sorting belong to that API, not to these local display fixtures.
		console.log('Explore cars request', prepareCatalogRequest(filters, pagination.page, pageSize));
	}

	function applyFilters(value: Partial<CatalogFilters>) {
		filters = { ...filters, ...value };
		currentPage = 1;
		requestCars();
	}

	async function changePage(value: number) {
		currentPage = getPagination(demoCatalogCars.length, value, pageSize).page;
		requestCars();
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

	<CatalogFilterBar {brands} onapply={applyFilters} />

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
					Showing {pagination.start}–{pagination.end} of {demoCatalogCars.length} cars
				</h2>
				<p class="mt-1 text-sm/6 text-ink-muted">Self-drive · Choose dates at checkout</p>
			</div>
			<div class="w-full sm:w-64">
				<SelectField
					name="sort"
					label="Sort by"
					options={sorts}
					bind:value={filters.sort}
					onchange={() => applyFilters({ sort: filters.sort })}
				/>
			</div>
		</div>
		<ul class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3" aria-label="Rental cars">
			{#each cars as car (car.id)}
				<li class="min-w-0"><CarCard {car} /></li>
			{/each}
		</ul>
		<div class="mt-8 flex flex-col items-center justify-between gap-4 sm:mt-10 sm:flex-row">
			<p class="text-sm/6 text-ink-muted">Page {pagination.page} of {pagination.totalPages}</p>
			<Pagination page={pagination.page} totalPages={pagination.totalPages} onchange={changePage} />
		</div>
	</section>
</div>
