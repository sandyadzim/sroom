<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { Plus } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { carUnitsEndpoint } from '#lib/api/catalog/car-unit.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import CarSelectField from '#lib/components/admin/CarSelectField.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import { carOptionsQuery } from '#lib/queries/car.ts';
	import type { CarUnit } from '#lib/types/catalog.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { carStatuses, carVariantName } from '#lib/utils/car.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const cars = createQuery(() => carOptionsQuery);
	const carId = $derived((page.shallow?.url ?? page.url).searchParams.get('car') ?? '');
	const car = $derived(cars.data?.find((item) => item.id === carId));
	const options = $derived(
		cars.data?.map((item) => ({
			value: item.id,
			label: carVariantName(item) + (item.status === 'inactive' ? ' · Inactive' : '')
		})) ?? []
	);
	const config: DataTableConfig<CarUnit> = $derived({
		endpoint: carUnitsEndpoint(carId),
		urlContext: { car: carId },
		label: 'Physical units',
		rowKey: (unit) => unit.id,
		columns: [
			{ label: 'Internal code', cell: code },
			{ label: 'Status', cell: status },
			{ label: 'Created', cell: created },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{
				name: 'status',
				label: 'Status',
				type: 'select',
				placeholder: 'All statuses',
				options: carStatuses
			}
		],
		pageSize: 20,
		emptyMessage: 'No physical units for this car variant.',
		filteredEmptyMessage: 'No units match this status.'
	});
</script>

<svelte:head><title>Physical units — SROOM</title></svelte:head>

<header class="mb-7 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Physical units</h1>
		<p class="mt-2 text-sm/6 text-ink-muted">Manage individual vehicles by car variant.</p>
	</div>
	<Button
		href={`${resolve('/admin/(staff)/units/new')}${car ? `?car=${encodeURIComponent(car.id)}` : ''}`}
	>
		<Plus size={18} strokeWidth={1.5} aria-hidden="true" /> Create unit
	</Button>
</header>

{#if cars.data?.length}
	<div class="mb-5 max-w-xl">
		<CarSelectField
			name="car"
			label="Car variant"
			placeholder="Select a car variant"
			{options}
			value={car?.id ?? ''}
			onchange={(id) =>
				goto(resolve(`/admin/(staff)/units?car=${encodeURIComponent(id)}`), {
					reset: false
				})}
		/>
	</div>
	{#if car}
		{#key car.id}<DataTableGenerator {config} />{/key}
	{:else}
		<p class="rounded-xl border border-line bg-page px-4 py-5 text-sm text-ink-muted" role="status">
			{carId
				? 'Car variant not found. Select another variant.'
				: 'Select a car variant to view its units.'}
		</p>
	{/if}
{:else if cars.isError}
	<p class="text-sm text-negative" role="alert">
		{getApiError(cars.error)?.message ?? 'Could not load car variants.'}
	</p>
	<Button
		class="mt-3"
		variant="secondary"
		disabled={cars.isFetching}
		onclick={() => cars.refetch()}
	>
		Try again
	</Button>
{:else if cars.isPending}
	<p class="text-sm text-ink-muted" role="status">Loading car variants…</p>
{:else}
	<p class="text-sm text-ink-muted">Create a car variant before adding physical units.</p>
	<Button class="mt-3" variant="secondary" href={resolve('/admin/(staff)/cars/new')}
		>Create car</Button
	>
{/if}

{#snippet code(unit: CarUnit)}
	<Button
		href={resolve('/admin/(staff)/units/[id]', { id: unit.id })}
		variant="ghost"
		size="custom"
		class="-ml-2 max-w-60 rounded-lg px-2 text-left wrap-anywhere underline decoration-primary/30 underline-offset-4"
		>{unit.internal_code}</Button
	>
{/snippet}

{#snippet status(unit: CarUnit)}
	<StatusBadge
		label={unit.status === 'active' ? 'Active' : 'Inactive'}
		tone={unit.status === 'active' ? 'success' : 'danger'}
	/>
{/snippet}

{#snippet created(unit: CarUnit)}
	<time datetime={unit.created_at} class="whitespace-nowrap"
		>{formatWitaDateTime(unit.created_at)}</time
	>
{/snippet}

{#snippet actions(unit: CarUnit)}
	<Button
		href={resolve('/admin/(staff)/units/[id]', { id: unit.id })}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View ${unit.internal_code}`}>View</Button
	>
{/snippet}
