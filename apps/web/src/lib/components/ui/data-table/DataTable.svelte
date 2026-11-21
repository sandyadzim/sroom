<script lang="ts" generics="Row extends object">
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import type { TableColumn } from '#lib/types/data-table.ts';
	import type { getPagination } from '#lib/utils/pagination.ts';

	let {
		label,
		rows,
		rowKey,
		columns,
		loading,
		failure,
		emptyMessage,
		pagination,
		total,
		pageSize,
		pageSizes,
		onPageSizeChange,
		onPageChange,
		onRetry
	}: {
		label: string;
		rows: Row[];
		rowKey: (row: Row) => string;
		columns: readonly TableColumn<Row>[];
		loading: boolean;
		failure?: { message: string; status?: number };
		emptyMessage: string;
		pagination: ReturnType<typeof getPagination>;
		total: number;
		pageSize: number;
		pageSizes: readonly number[];
		onPageSizeChange: (size: number) => void;
		onPageChange: (page: number) => void;
		onRetry: () => void;
	} = $props();

	let viewportWidth = $state(0);
	const messageWidth = $derived(viewportWidth ? `${viewportWidth}px` : undefined);

	const sizeOptions = $derived(
		pageSizes.map((size) => ({ value: String(size), label: String(size) }))
	);
	const pageButtonClass =
		'border border-line aria-disabled:cursor-not-allowed aria-disabled:text-ink-muted aria-disabled:hover:bg-transparent';
	const stickyColumnClass =
		'sticky right-0 z-10 before:pointer-events-none before:absolute before:inset-y-0 before:left-0 before:w-px before:bg-line';
	const lastColumn = $derived(columns.at(-1));
	const stickyColumn = $derived(lastColumn?.sticky === 'right' ? lastColumn : undefined);
	const previousUnavailable = $derived(loading || Boolean(failure) || pagination.page <= 1);
	const nextUnavailable = $derived(
		loading || Boolean(failure) || pagination.page >= pagination.totalPages
	);
</script>

<div class="min-w-0 rounded-xl border border-line bg-panel">
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (keyboard users can scroll the table) -->
	<div
		role="region"
		aria-busy={loading}
		aria-label={`${label} table`}
		bind:clientWidth={viewportWidth}
		tabindex="0"
		class="overflow-x-auto rounded-t-xl focus-visible:-outline-offset-2"
	>
		<table class="w-full min-w-max border-collapse text-left text-sm">
			<caption class="sr-only">{label}</caption>
			<thead class="border-b border-line bg-page">
				<tr>
					{#each columns as column (column.label)}
						<th
							scope="col"
							class={[
								'px-4 py-3.5 font-semibold whitespace-nowrap sm:px-5',
								column === stickyColumn && [stickyColumnClass, 'bg-page']
							]}
						>
							{column.label}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody class="divide-y divide-line">
				{#if failure}
					<tr>
						<td colspan={columns.length}>
							<div class="sticky left-0 px-4 py-8 sm:px-5" style:width={messageWidth}>
								<p role="alert" class="font-medium text-negative">{failure.message}</p>
								{#if failure.status !== 401 && failure.status !== 403}
									<Button variant="secondary" class="mt-3" onclick={onRetry}>Try again</Button>
								{/if}
							</div>
						</td>
					</tr>
				{:else if loading}
					{#each [0, 1, 2] as index (index)}
						<tr aria-hidden="true">
							{#each columns as column (column.label)}
								<td
									class={[
										'px-4 py-4 sm:px-5',
										column === stickyColumn && [stickyColumnClass, 'bg-panel']
									]}
								>
									<div class="h-4 w-16 rounded bg-panel-muted motion-safe:animate-pulse"></div>
								</td>
							{/each}
						</tr>
					{/each}
				{:else if rows.length === 0}
					<tr>
						<td colspan={columns.length}>
							<p
								class="sticky left-0 px-4 py-10 text-center text-ink-muted sm:px-5"
								style:width={messageWidth}
							>
								{emptyMessage}
							</p>
						</td>
					</tr>
				{:else}
					{#each rows as row (rowKey(row))}
						<tr>
							{#each columns as column (column.label)}
								<td
									class={[
										'max-w-80 px-4 py-3.5 align-middle wrap-anywhere sm:px-5',
										column === stickyColumn && [stickyColumnClass, 'bg-panel']
									]}
								>
									{#if column.cell}
										{@render column.cell(row)}
									{:else}
										{row[column.field]}
									{/if}
								</td>
							{/each}
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<div
		class="flex flex-wrap items-center justify-end gap-x-4 gap-y-2 border-t border-line px-3 py-2.5 sm:px-5"
	>
		<p role="status" class="text-sm text-ink-muted tabular-nums">
			{#if loading}
				Loading results…
			{:else if !failure}
				{total.toLocaleString('en-US')} {total === 1 ? 'result' : 'results'}
			{/if}
		</p>
		<div class="flex items-center gap-2">
			<div class="w-20">
				<SelectField
					label="Rows per page"
					name="page-size"
					value={String(pageSize)}
					options={sizeOptions}
					onchange={(value) => onPageSizeChange(Number(value))}
					disabled={failure?.status === 401 || failure?.status === 403}
					compact
				/>
			</div>
			<nav aria-label={`${label} pagination`} class="flex items-center gap-1">
				<Button
					variant="ghost"
					size="icon"
					class={pageButtonClass}
					aria-label="Previous page"
					aria-disabled={previousUnavailable}
					onclick={() => !previousUnavailable && onPageChange(pagination.page - 1)}
				>
					<ChevronLeft size={18} strokeWidth={1.5} aria-hidden="true" />
				</Button>
				<span class="px-2 text-center text-sm whitespace-nowrap tabular-nums">
					<span class="sr-only">Page </span>{pagination.page} of {pagination.totalPages}
				</span>
				<Button
					variant="ghost"
					size="icon"
					class={pageButtonClass}
					aria-label="Next page"
					aria-disabled={nextUnavailable}
					onclick={() => !nextUnavailable && onPageChange(pagination.page + 1)}
				>
					<ChevronRight size={18} strokeWidth={1.5} aria-hidden="true" />
				</Button>
			</nav>
		</div>
	</div>
</div>
