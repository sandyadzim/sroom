import { staffApi } from './staff';
import type { TablePage, TableParams } from '#lib/types/data-table.ts';

export async function fetchTablePage<Row>(
	endpoint: string,
	{ page, limit, filters }: TableParams,
	signal?: AbortSignal
) {
	const { data } = await staffApi.get<TablePage<Row>>(endpoint, {
		params: { ...filters, page, limit },
		signal
	});
	return data;
}
