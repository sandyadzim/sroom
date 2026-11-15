import { queryOptions } from '@tanstack/svelte-query';
import { fetchCar } from '#lib/api/catalog/car.ts';

export function carDetailQuery(id: string) {
	return queryOptions({
		queryKey: ['catalog-car', id],
		queryFn: ({ signal }) => fetchCar(id, signal),
		refetchOnWindowFocus: false
	});
}
