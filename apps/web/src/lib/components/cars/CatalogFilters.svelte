<script lang="ts">
	import { Search, SlidersHorizontal } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import type { CatalogFilters } from '#lib/types/catalog.ts';

	let {
		brands,
		onapply
	}: { brands: string[]; onapply: (filters: Omit<CatalogFilters, 'sort'>) => void } = $props();
	const initialFilters = { search: '', brand: '', seats: '', price: '' };
	let filters = $state({ ...initialFilters });
	let expanded = $state(false);
	const filterId = $props.id();
	const activeCount = $derived(
		[filters.brand, filters.seats, filters.price].filter(Boolean).length
	);
	const hasFilters = $derived(Object.values(filters).some(Boolean));
	const brandOptions = $derived([
		{ value: '', label: 'All brands' },
		...brands.map((brand) => ({ value: brand, label: brand }))
	]);
	const seats = [
		{ value: '', label: 'Any number' },
		{ value: '4', label: '4 seats' },
		{ value: '5', label: '5 seats' },
		{ value: '7', label: '7 seats' }
	];
	const prices = [
		{ value: '', label: 'Any daily rate' },
		{ value: 'under-400', label: 'Up to IDR 400,000' },
		{ value: '400-700', label: 'IDR 400,001–700,000' },
		{ value: 'over-700', label: 'Above IDR 700,000' }
	];

	function apply() {
		onapply({ ...filters });
	}
	function reset() {
		filters = { ...initialFilters };
		apply();
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
	<div class="flex flex-col items-start gap-3 lg:block">
		<div class="flex w-full min-w-0 items-end gap-2 sm:gap-3">
			<div class="min-w-0 flex-1">
				<TextField
					name="search"
					type="search"
					label="Find your car"
					placeholder="Search brand or model"
					bind:value={filters.search}
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
			'mt-4 gap-4 sm:grid-cols-3 lg:mt-6 lg:grid lg:grid-cols-[1fr_1fr_1.25fr_auto] lg:items-end lg:gap-6'
		]}
	>
		<SelectField
			name="brand"
			label="Brand"
			options={brandOptions}
			bind:value={filters.brand}
			onchange={apply}
		/>
		<SelectField
			name="seats"
			label="Seats"
			options={seats}
			bind:value={filters.seats}
			onchange={apply}
		/>
		<SelectField
			name="price"
			label="Price per day"
			options={prices}
			bind:value={filters.price}
			onchange={apply}
		/>
		<Button
			variant="ghost"
			disabled={!hasFilters}
			class="sm:col-span-3 lg:col-span-1"
			onclick={reset}>Reset filters</Button
		>
	</div>
</form>
