<script lang="ts">
	import { brandsEndpoint } from '#lib/api/catalog/brand.ts';
	import BrandEditor from '#lib/components/admin/BrandEditor.svelte';
	import ImagePreviewDialog from '#lib/components/ui/ImagePreviewDialog.svelte';
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
		{@const characters = Array.from(filename)}
		{@const suffix = characters.length > 22 ? characters.slice(-10).join('') : ''}
		<ImagePreviewDialog title="Brand Image" {filename} src={brand.image_url}>
			{#snippet trigger({ props })}
				<button
					{...props}
					type="button"
					title={filename}
					aria-label={filename}
					class="flex max-w-40 min-w-0 cursor-pointer rounded-sm text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary sm:max-w-48"
				>
					<span class="truncate">{suffix ? characters.slice(0, -10).join('') : filename}</span>
					{#if suffix}<span class="shrink-0 whitespace-pre">{suffix}</span>{/if}
				</button>
			{/snippet}
		</ImagePreviewDialog>
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
