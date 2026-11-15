<script lang="ts">
	import { resolve } from '$app/paths';
	import { Plus } from '@lucide/svelte';
	import { carsEndpoint } from '#lib/api/catalog/car.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import ImagePreviewLink from '#lib/components/ui/ImagePreviewLink.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { AdminCar } from '#lib/types/catalog.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { carCategories, carOptionLabel, carStatuses, carTransmissions } from '#lib/utils/car.ts';
	import { formatIDR } from '#lib/utils/currency.ts';

	const config: DataTableConfig<AdminCar> = {
		endpoint: carsEndpoint,
		label: 'Cars',
		rowKey: (car) => car.id,
		columns: [
			{ label: 'Car', cell: vehicle },
			{ label: 'Image', cell: image },
			{ label: 'Category', cell: category },
			{ label: 'Seats', field: 'seats' },
			{ label: 'Transmission', cell: transmission },
			{ label: 'Daily rate', cell: rate },
			{ label: 'Status', cell: status },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{
				name: 'search',
				label: 'Search',
				type: 'search',
				placeholder: 'Search brand, model, or trim'
			},
			{
				name: 'status',
				label: 'Status',
				type: 'select',
				placeholder: 'All statuses',
				options: carStatuses
			},
			{
				name: 'category',
				label: 'Category',
				type: 'select',
				placeholder: 'All categories',
				options: carCategories
			},
			{
				name: 'transmission',
				label: 'Transmission',
				type: 'select',
				placeholder: 'All transmissions',
				options: carTransmissions
			}
		],
		pageSize: 20,
		emptyMessage: 'No cars yet.',
		filteredEmptyMessage: 'No cars match your filters.'
	};
</script>

<svelte:head><title>Cars — SROOM</title></svelte:head>

<header class="mb-7 flex flex-wrap items-center justify-between gap-3">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Cars</h1>
	<Button href={resolve('/admin/(staff)/cars/new')}>
		<Plus size={18} strokeWidth={1.5} aria-hidden="true" /> Create car
	</Button>
</header>

<DataTableGenerator {config} />

{#snippet vehicle(car: AdminCar)}
	<Link
		href={resolve('/admin/(staff)/cars/[id]', { id: car.id })}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
		>{car.brand_name} {car.model}</Link
	>
	{#if car.trim}<p class="text-xs text-ink-muted">{car.trim}</p>{/if}
{/snippet}

{#snippet image(car: AdminCar)}
	{#if car.cover}
		<ImagePreviewLink
			title="Car Image"
			label={car.cover.label || 'Cover photo'}
			alt={car.cover.alt_text}
			src={car.cover.url}
		/>
	{:else}
		<span class="text-ink-muted">—</span>
	{/if}
{/snippet}

{#snippet category(car: AdminCar)}{carOptionLabel(carCategories, car.category)}{/snippet}
{#snippet transmission(car: AdminCar)}{carOptionLabel(carTransmissions, car.transmission)}{/snippet}
{#snippet rate(car: AdminCar)}
	<span class="font-medium whitespace-nowrap tabular-nums">{formatIDR(car.daily_rate_idr)}</span>
{/snippet}
{#snippet status(car: AdminCar)}
	<StatusBadge
		label={carOptionLabel(carStatuses, car.status)}
		tone={car.status === 'active' ? 'success' : 'danger'}
	/>
{/snippet}
{#snippet actions(car: AdminCar)}
	<Button
		href={resolve('/admin/(staff)/cars/[id]', { id: car.id })}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View ${[car.brand_name, car.model, car.trim].filter(Boolean).join(' ')}`}
		>View</Button
	>
{/snippet}
