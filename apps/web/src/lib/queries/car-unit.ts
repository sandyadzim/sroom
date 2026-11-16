import { queryOptions } from '@tanstack/svelte-query';
import { fetchCarUnit } from '#lib/api/catalog/car-unit.ts';

export function carUnitQuery(id: string) {
	return queryOptions({
		queryKey: ['catalog-car-unit', id],
		queryFn: ({ signal }) => fetchCarUnit(id, signal),
		refetchOnWindowFocus: false
	});
}
