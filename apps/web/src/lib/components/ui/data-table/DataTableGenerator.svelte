<script lang="ts" generics="Row extends object">
	import { Search } from '@lucide/svelte';
	import AdminCarFilter from '#lib/components/admin/AdminCarFilter.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { createDataTable } from '#lib/queries/createDataTable.svelte.ts';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import DataTable from './DataTable.svelte';

	let { config }: { config: DataTableConfig<Row> } = $props();
	const table = createDataTable(() => config);
	const hasFilters = $derived(Object.keys(table.filters).length > 0);
	const searchFilters = $derived(
		config.filters?.filter((filter) => filter.type === 'search') ?? []
	);
	const controlFilters = $derived(
		config.filters?.filter((filter) => filter.type !== 'search') ?? []
	);
	const filterRows = $derived(
		Array.from({ length: Math.ceil(controlFilters.length / 3) }, (_, index) =>
			controlFilters.slice(index * 3, index * 3 + 3)
		)
	);
	const gridColumns = ['sm:grid-cols-1', 'sm:grid-cols-2', 'sm:grid-cols-3'];
</script>

{#if config.filters?.length}
	<div class="mb-5 grid gap-4">
		{#each searchFilters as filter (filter.name)}
			<TextField
				type="search"
				name={filter.name}
				label={filter.label}
				placeholder={filter.placeholder}
				value={table.searchValue(filter.name)}
				oninput={(event) =>
					table.updateSearch(
						filter.name,
						event.currentTarget.value,
						event instanceof InputEvent && event.isComposing
					)}
				oncompositionend={(event) => table.updateSearch(filter.name, event.currentTarget.value)}
				onclear={() => table.clearSearch(filter.name)}
				autocomplete="off"
			>
				{#snippet leading()}
					<Search size={18} strokeWidth={1.5} aria-hidden="true" />
				{/snippet}
			</TextField>
		{/each}

		{#each filterRows as row (row[0].name)}
			<div class={['grid gap-4', gridColumns[row.length - 1]]}>
				{#each row as filter (filter.name)}
					{#if filter.type === 'date'}
						<TextField
							type="date"
							label={filter.label}
							name={filter.name}
							value={table.filters[filter.name] ?? ''}
							onchange={(event) => table.setFilter(filter.name, event.currentTarget.value)}
							onclear={() => table.setFilter(filter.name, '')}
						/>
					{:else if filter.type === 'car'}
						<AdminCarFilter
							name={filter.name}
							label={filter.label}
							value={table.filters[filter.name] ?? ''}
							placeholder={filter.placeholder}
							onchange={(value) => table.setFilter(filter.name, value)}
						/>
					{:else}
						<SelectField
							label={filter.label}
							name={filter.name}
							placeholder={filter.placeholder}
							options={filter.options}
							value={table.filters[filter.name] ?? ''}
							onchange={(value) => table.setFilter(filter.name, value)}
							clearable
						/>
					{/if}
				{/each}
			</div>
		{/each}
	</div>
{/if}

<DataTable
	label={config.label}
	rows={table.query.data?.data ?? []}
	rowKey={config.rowKey}
	columns={config.columns}
	loading={!table.failure && (table.query.isPending || table.page > table.pagination.totalPages)}
	failure={table.failure}
	emptyMessage={hasFilters
		? (config.filteredEmptyMessage ?? 'No results match your filters.')
		: config.emptyMessage}
	pagination={table.pagination}
	total={table.query.data?.page.total ?? 0}
	pageSize={table.pageSize}
	pageSizes={table.pageSizes}
	onPageChange={table.changePage}
	onPageSizeChange={table.changePageSize}
	onRetry={table.refetch}
/>
