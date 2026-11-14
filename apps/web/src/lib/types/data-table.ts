import type { Snippet } from 'svelte';

export type TablePage<Row> = {
	data: Row[];
	page: { number: number; limit: number; total: number };
};

export type TableParams = { page: number; limit: number; filters: Record<string, string> };

export type TableFilter = {
	name: string;
	label: string;
} & (
	| { type: 'search'; placeholder?: string }
	| { type: 'select'; placeholder?: string; options: readonly { value: string; label: string }[] }
);

export type TableColumn<Row> = { label: string } & (
	{ field: Extract<keyof Row, string>; cell?: never } | { field?: never; cell: Snippet<[Row]> }
);

export type DataTableConfig<Row> = {
	endpoint: string;
	label: string;
	rowKey: (row: Row) => string;
	columns: readonly TableColumn<Row>[];
	filters?: readonly TableFilter[];
	pageSize: number;
	pageSizes?: readonly number[];
	emptyMessage: string;
	filteredEmptyMessage?: string;
};
