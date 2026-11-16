<script lang="ts">
	import { RotateCcw, Search, SlidersHorizontal } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import type { CatalogFilters, PublicBrand } from '#lib/types/catalog.ts';
	import { carCategories, carTransmissions } from '#lib/utils/car.ts';
	import {
		catalogPrices,
		defaultCatalogFilters,
		emptyCatalogFilters
	} from '#lib/utils/catalog-filters.ts';

	type FilterValues = Omit<CatalogFilters, 'sort'>;

	let {
		brands,
		brandsUnavailable,
		filters,
		onapply
	}: {
		brands: PublicBrand[];
		brandsUnavailable: boolean;
		filters: CatalogFilters;
		onapply: (changes: Partial<FilterValues>) => void;
	} = $props();
	let search = $derived(filters.search);
	let expanded = $state(false);
	const filterId = $props.id();
	const commonSeats = ['4', '5', '7'];
	const activeCount = $derived(
		[filters.brand, filters.category, filters.seats, filters.transmission, filters.price].filter(
			Boolean
		).length
	);
	const seatOptions = $derived([
		{ value: '', label: 'Any number' },
		...commonSeats.map((value) => ({ value, label: `${value} seats` })),
		...(filters.seats && !commonSeats.includes(filters.seats)
			? [
					{
						value: filters.seats,
						label: Number(filters.seats) === 1 ? '1 seat' : `${filters.seats} seats`
					}
				]
			: [])
	]);
	const fields = $derived([
		{
			name: 'brand',
			label: 'Brand',
			disabled: brandsUnavailable,
			options: [
				{ value: '', label: brandsUnavailable ? 'Brands unavailable' : 'All brands' },
				...brands.map(({ id, name }) => ({ value: id, label: name })),
				...(filters.brand && !brands.some(({ id }) => id === filters.brand)
					? [{ value: filters.brand, label: 'Selected brand' }]
					: [])
			]
		},
		{
			name: 'category',
			label: 'Category',
			options: [{ value: '', label: 'All categories' }, ...carCategories]
		},
		{ name: 'seats', label: 'Seats', options: seatOptions },
		{
			name: 'transmission',
			label: 'Transmission',
			options: [{ value: '', label: 'All transmissions' }, ...carTransmissions]
		},
		{
			name: 'price',
			label: 'Price per day',
			options: [
				{ value: '', label: 'Any daily rate' },
				...catalogPrices.filter(({ value, legacy }) => !legacy || value === filters.price)
			]
		}
	] as const);

	function apply(changes: Partial<FilterValues> = {}) {
		onapply({ search: search.trim(), ...changes });
	}

	function reset() {
		search = '';
		onapply(emptyCatalogFilters);
	}
</script>

<form
	role="search"
	aria-label="Find a rental car"
	onsubmit={(event) => {
		event.preventDefault();
		apply();
	}}
	class="rounded-panel bg-panel-muted p-4 sm:p-6"
>
	{#if filters.sort !== defaultCatalogFilters.sort}
		<input type="hidden" name="sort" value={filters.sort} />
	{/if}
	<div class="flex flex-col items-start gap-3 lg:block">
		<div class="flex w-full min-w-0 items-end gap-2 sm:gap-3">
			<div class="min-w-0 flex-1">
				<TextField
					name="search"
					type="search"
					label="Find your car"
					placeholder="Brand, model, or trim"
					bind:value={search}
					onclear={() => apply({ search: '' })}
				/>
			</div>
			<Button
				type="submit"
				size="custom"
				class="min-h-13 shrink-0 rounded-xl px-4 sm:px-5"
				aria-label="Search cars"
				><Search size={20} strokeWidth={1.5} aria-hidden="true" /><span class="hidden sm:inline"
					>Search</span
				></Button
			>
		</div>
		<div class="lg:hidden">
			<Button
				variant="ghost"
				size="custom"
				class="rounded-xl px-3"
				aria-expanded={expanded}
				aria-controls={filterId}
				onclick={() => (expanded = !expanded)}
				><SlidersHorizontal
					size={18}
					strokeWidth={1.5}
					aria-hidden="true"
				/>Filters{#if activeCount}<span>({activeCount})</span>{/if}</Button
			>
		</div>
	</div>
	<div
		id={filterId}
		class={[
			expanded ? 'grid' : 'hidden',
			'mt-4 gap-4 sm:grid-cols-2 lg:mt-6 lg:grid lg:grid-cols-3 lg:items-end lg:gap-6'
		]}
	>
		{#each fields as field (field.name)}
			<SelectField
				name={field.name}
				label={field.label}
				options={field.options}
				clearable
				value={filters[field.name]}
				disabled={'disabled' in field && field.disabled}
				onchange={(value) => apply({ [field.name]: value })}
			/>
		{/each}
		<Button
			variant="ghost"
			disabled={!search && !filters.search && !activeCount}
			class="gap-2 justify-self-end border border-line hover:border-ink-muted/50 lg:mb-1"
			onclick={reset}
		>
			<RotateCcw size={16} strokeWidth={1.5} aria-hidden="true" />
			Reset filters
		</Button>
	</div>
</form>
