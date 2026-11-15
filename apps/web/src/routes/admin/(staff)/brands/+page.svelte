<script lang="ts">
	import { brandsEndpoint } from '#lib/api/catalog/brand.ts';
	import BrandEditor from '#lib/components/admin/BrandEditor.svelte';
	import ImagePreviewLink from '#lib/components/ui/ImagePreviewLink.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { Brand } from '#lib/types/catalog.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const config: DataTableConfig<Brand> = {
		endpoint: brandsEndpoint,
		label: 'Brands',
		rowKey: (brand) => brand.id,
		columns: [
			{ label: 'Name', cell: brandName },
			{ label: 'Status', cell: status },
			{ label: 'Image', cell: image },
			{ label: 'Last updated', cell: updatedAt },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{ name: 'search', label: 'Search', type: 'search', placeholder: 'Search by brand name' }
		],
		pageSize: 20,
		emptyMessage: 'No brands yet.',
		filteredEmptyMessage: 'No brands match your search.'
	};
</script>

<svelte:head>
	<title>Brands — SROOM</title>
</svelte:head>

<div class="mb-7 flex flex-wrap items-center justify-between gap-3">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Brands</h1>
	<BrandEditor />
</div>

<DataTableGenerator {config} />

{#snippet brandName(brand: Brand)}
	<span class="font-medium">{brand.name}</span>
{/snippet}

{#snippet status(brand: Brand)}
	{@const active = brand.status === 'active'}
	<StatusBadge label={active ? 'Active' : 'Inactive'} tone={active ? 'success' : 'danger'} />
{/snippet}

{#snippet image(brand: Brand)}
	{#if brand.image_object_key}
		{@const filename =
			brand.image_filename || brand.image_object_key.split('/').at(-1) || 'Brand image'}
		<ImagePreviewLink title="Brand Image" label={filename} src={brand.image_url} />
	{:else}
		<span class="text-ink-muted">—</span>
	{/if}
{/snippet}

{#snippet updatedAt(brand: Brand)}
	<time datetime={brand.updated_at} class="whitespace-nowrap tabular-nums">
		{formatWitaDateTime(brand.updated_at)}
	</time>
{/snippet}

{#snippet actions(brand: Brand)}
	<BrandEditor {brand} />
{/snippet}
