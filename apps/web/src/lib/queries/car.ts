import { queryOptions } from '@tanstack/svelte-query';
import { carsEndpoint, fetchActiveCarsForSelect, fetchCar } from '#lib/api/catalog/car.ts';

export const activeCarOptionsQuery = queryOptions({
	queryKey: [carsEndpoint, 'options', 'active'],
	queryFn: ({ signal }) => fetchActiveCarsForSelect(signal)
});

export function carDetailQuery(id: string) {
	return queryOptions({
		queryKey: ['catalog-car', id],
		queryFn: ({ signal }) => fetchCar(id, signal),
		refetchOnWindowFocus: false
	});
}
